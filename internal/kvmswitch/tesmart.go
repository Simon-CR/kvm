package kvmswitch

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"kvm/internal/logging"
)

var logger = logging.GetSubsystemLogger("kvmswitch")

type TESmartSwitch struct {
	mu           sync.Mutex
	ip           string
	port         int
	activePort   int
	pollInterval time.Duration
	cancelPoller context.CancelFunc
	enabled      bool
}

var (
	cmdHeader   = []byte{0xAA, 0xBB, 0x03}
	cmdFooter   = byte(0xEE)
	cmdSwitch   = byte(0x01)
	cmdQuery    = byte(0x10)
)

var (
	GlobalSwitch *TESmartSwitch
)

func NewTESmartSwitch() *TESmartSwitch {
	return &TESmartSwitch{
		activePort: -1,
	}
}

func (s *TESmartSwitch) UpdateConfig(enabled bool, ip string, port int, pollIntervalSec int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = enabled
	s.ip = ip
	s.port = port
	s.pollInterval = time.Duration(pollIntervalSec) * time.Second

	if s.cancelPoller != nil {
		s.cancelPoller()
		s.cancelPoller = nil
	}

	if s.enabled && s.ip != "" {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelPoller = cancel
		go s.pollLoop(ctx)
	}
}

func (s *TESmartSwitch) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Initial poll
	s.QueryActivePort()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.QueryActivePort()
		}
	}
}

func (s *TESmartSwitch) executeCommand(cmd []byte, expectResponse bool) ([]byte, error) {
	address := fmt.Sprintf("%s:%d", s.ip, s.port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write(cmd)
	if err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}

	if !expectResponse {
		return nil, nil
	}

	resp := make([]byte, 6)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	return resp, nil
}

func (s *TESmartSwitch) SwitchPort(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return fmt.Errorf("switch is disabled")
	}

	cmd := []byte{cmdHeader[0], cmdHeader[1], cmdHeader[2], cmdSwitch, byte(port), cmdFooter}
	_, err := s.executeCommand(cmd, false)
	if err != nil {
		return err
	}
	
	s.activePort = port
	return nil
}

func (s *TESmartSwitch) QueryActivePort() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return -1, fmt.Errorf("switch is disabled")
	}

	cmd := []byte{cmdHeader[0], cmdHeader[1], cmdHeader[2], cmdQuery, 0x00, cmdFooter}
	resp, err := s.executeCommand(cmd, true)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to query active port")
		return -1, err
	}

	if len(resp) >= 6 && resp[0] == 0xAA && resp[1] == 0xBB {
		port := int(resp[4])
		s.activePort = port
		return port, nil
	}
	
	return -1, fmt.Errorf("invalid response length or header: %x", resp)
}

func (s *TESmartSwitch) GetActivePort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activePort
}

func InitSwitch() {
	GlobalSwitch = NewTESmartSwitch()
}

func TestConnection(ip string, port int) (int, time.Duration, error) {
	start := time.Now()
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return -1, 0, fmt.Errorf("connect error: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	cmd := []byte{cmdHeader[0], cmdHeader[1], cmdHeader[2], cmdQuery, 0x00, cmdFooter}
	_, err = conn.Write(cmd)
	if err != nil {
		return -1, 0, fmt.Errorf("write error: %w", err)
	}

	resp := make([]byte, 6)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return -1, 0, fmt.Errorf("read error: %w", err)
	}

	latency := time.Since(start)

	if len(resp) >= 6 && resp[0] == 0xAA && resp[1] == 0xBB {
		activePort := int(resp[4])
		return activePort, latency, nil
	}
	return -1, 0, fmt.Errorf("invalid response length or header: %x", resp)
}

func (s *TESmartSwitch) executeASCIICommand(cmd string, expectResponse bool) (string, error) {
	address := fmt.Sprintf("%s:%d", s.ip, s.port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("connect error: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}

	if !expectResponse {
		return "", nil
	}

	resp := make([]byte, 256)
	n, err := conn.Read(resp)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	return string(resp[:n]), nil
}

func (s *TESmartSwitch) QueryNetworkInfo() (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return "", "", fmt.Errorf("switch is disabled")
	}

	ipResp, err := s.executeASCIICommand("IP?\r\n", true)
	if err != nil {
		return "", "", err
	}

	gwResp, err := s.executeASCIICommand("GW?\r\n", true)
	if err != nil {
		return "", "", err
	}
	
	// Parse Responses like "IP:192.168.001.010;"
	var ip, gw string
	if len(ipResp) > 3 && ipResp[:3] == "IP:" {
		ip = ipResp[3:]
		if ip[len(ip)-1] == ';' {
			ip = ip[:len(ip)-1]
		}
	} else {
		return "", "", fmt.Errorf("invalid IP response: %s", ipResp)
	}

	if len(gwResp) > 3 && gwResp[:3] == "GW:" {
		gw = gwResp[3:]
		if gw[len(gw)-1] == ';' {
			gw = gw[:len(gw)-1]
		}
	} else {
		return "", "", fmt.Errorf("invalid GW response: %s", gwResp)
	}

	return ip, gw, nil
}

func (s *TESmartSwitch) ConfigureNetwork(newIP, newGateway string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return fmt.Errorf("switch is disabled")
	}

	_, err := s.executeASCIICommand(fmt.Sprintf("IP:%s;\r\n", newIP), false)
	if err != nil {
		return fmt.Errorf("failed to set IP: %w", err)
	}

	_, err = s.executeASCIICommand(fmt.Sprintf("GW:%s;\r\n", newGateway), false)
	if err != nil {
		return fmt.Errorf("failed to set GW: %w", err)
	}

	return nil
}


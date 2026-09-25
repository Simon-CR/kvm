import React, { useEffect, useState } from 'react';
import { Select } from 'antd';
import { useUiStore } from '@/hooks/stores';

export interface KvmSwitchConfig {
  enabled: boolean;
  switch_ip: string;
  switch_port: number;
  total_ports: number;
  port_names: Record<string, string>;
  hotkeys_enabled: boolean;
  poll_interval_sec: number;
}

export const KvmSwitchSelector: React.FC = () => {
  const [config, setConfig] = useState<KvmSwitchConfig | null>(null);
  const [activePort, setActivePort] = useState<number>(-1);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const res = await fetch('/api/kvm-switch/config');
        if (res.ok) {
          const data = await res.json();
          setConfig(data);
        }
      } catch (e) {
        console.error('Failed to fetch KVM switch config', e);
      }
    };
    fetchConfig();
  }, []);

  useEffect(() => {
    if (!config?.enabled) return;
    const fetchStatus = async () => {
      try {
        const res = await fetch('/api/kvm-switch/status');
        if (res.ok) {
          const data = await res.json();
          setActivePort(data.active_port);
        }
      } catch (e) {
        console.error('Failed to fetch KVM switch status', e);
      }
    };
    fetchStatus();
    const interval = setInterval(fetchStatus, (config.poll_interval_sec || 5) * 1000);
    return () => clearInterval(interval);
  }, [config]);

  useEffect(() => {
    if (!config?.hotkeys_enabled) return;
    const handleKeyDown = async (e: KeyboardEvent) => {
      if (e.ctrlKey && e.altKey) {
        const keyMap: Record<string, number> = {
          '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8
        };
        const port = keyMap[e.key];
        if (port && port <= config.total_ports) {
          handleSelectPort(port);
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [config]);

  const handleSelectPort = async (port: number) => {
    try {
      const res = await fetch('/api/kvm-switch/select', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port })
      });
      if (res.ok) {
        setActivePort(port);
      }
    } catch (e) {
      console.error('Failed to select port', e);
    }
  };

  const setDisableFocusTrap = useUiStore(state => state.setDisableVideoFocusTrap);

  if (!config?.enabled) {
    return null;
  }

  const ports = Array.from({ length: config.total_ports }, (_, i) => i + 1);
  
  const options = ports.map(port => ({
    value: port,
    label: config.port_names?.[port.toString()] || `Port ${port}`
  }));

  return (
    <div 
      className="inline-flex items-center px-1"
      style={{ zIndex: 100 }}
      onMouseDown={(e) => {
        e.stopPropagation();
        setDisableFocusTrap(true);
      }}
      onClick={(e) => {
        e.stopPropagation();
        setDisableFocusTrap(true);
      }}
    >
      <Select
        placeholder="Select KVM Port"
        getPopupContainer={(triggerNode) => triggerNode.parentElement || document.body}
        dropdownStyle={{ zIndex: 9999 }}
        onDropdownVisibleChange={(open) => {
          if (open) {
            setDisableFocusTrap(true);
          }
        }}
        value={activePort > 0 ? activePort : undefined}
        onChange={(val: number) => handleSelectPort(val)}
        style={{ width: '160px' }}
        options={options}
      />
    </div>
  );
};

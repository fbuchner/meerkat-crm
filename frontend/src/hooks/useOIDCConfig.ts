import { useState, useEffect } from 'react';
import { API_BASE_URL } from '../auth';

interface OIDCConfig {
  enabled: boolean;
  provider_name: string;
}

export function useOIDCConfig(): OIDCConfig {
  const [config, setConfig] = useState<OIDCConfig>({ enabled: false, provider_name: 'SSO' });

  useEffect(() => {
    fetch(`${API_BASE_URL}/auth/oidc/config`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (data) {
          setConfig({
            enabled: data.enabled === true,
            provider_name: data.provider_name || 'SSO',
          });
        }
      })
      .catch(() => {
        // OIDC config is best-effort; if it fails, SSO button stays hidden
      });
  }, []);

  return config;
}

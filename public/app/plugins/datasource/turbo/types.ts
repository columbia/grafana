import { SQLOptions } from 'app/features/plugins/sql/types';

// import { DataSourceJsonData } from '@grafana/data';

export enum TurboTLSModes {
  disable = 'disable',
  require = 'require',
  verifyCA = 'verify-ca',
  verifyFull = 'verify-full',
}

export enum TurboTLSMethods {
  filePath = 'file-path',
  fileContent = 'file-content',
}

// export interface TurboOptions extends DataSourceJsonData {}

export interface TurboOptions extends SQLOptions {
  tlsConfigurationMethod?: TurboTLSMethods;
  sslmode?: TurboTLSModes;
  sslRootCertFile?: string;
  sslCertFile?: string;
  sslKeyFile?: string;
  turboVersion?: number;
  timescaledb?: boolean;
  enableSecureSocksProxy?: boolean;
}

export interface SecureJsonData {
  password: string;
}

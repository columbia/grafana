// import { Dispatch, SetStateAction, useState } from 'react';
// import { useDeepCompareEffect } from 'react-use';

// import {
//   DataSourcePluginOptionsEditorProps,
//   DataSourceSettings,
//   SelectableValue,
//   updateDatasourcePluginJsonDataOption,
//   updateDatasourcePluginOption,
// } from '@grafana/data';
// import { getBackendSrv } from '@grafana/runtime';
// import { getDatasourceSrv } from 'app/features/plugins/datasource_srv';

// import { TurboDatasource } from '../datasource';
// import { TurboOptions, TurboTLSModes, SecureJsonData } from '../types';

// // import { turboVersions } from './ConfigurationEditor';

// type Options = {
//   props: DataSourcePluginOptionsEditorProps<TurboOptions, SecureJsonData>;
//   setVersionOptions: Dispatch<SetStateAction<Array<SelectableValue<number>>>>;
// };

// export function useAutoDetectFeatures({ props, setVersionOptions }: Options) {
//   const [saved, setSaved] = useState(false);
//   const { options, onOptionsChange } = props;

//   useDeepCompareEffect(() => {
//     const getVersion = async () => {
//       if (!saved) {
//         // We need to save the datasource before we can get the version so we can query the database with the options we have.
//         const result = await getBackendSrv().put<{ datasource: DataSourceSettings }>(
//           `/api/datasources/${options.id}`,
//           options
//         );

//         setSaved(true);
//         // This is needed or else we get an error when we try to save the datasource.
//         updateDatasourcePluginOption({ options, onOptionsChange }, 'version', result.datasource.version);
//       } else {
//         const datasource = await getDatasourceSrv().loadDatasource(options.name);

//       //   if (datasource instanceof TurboDatasource) {
//       //     const version = await datasource.getVersion();
//       //     const versionNumber = parseInt(version, 10);

//       //     // timescaledb is only available for 9.6+
//       //     if (versionNumber >= 906 && !options.jsonData.timescaledb) {
//       //       const timescaledbVersion = await datasource.getTimescaleDBVersion();
//       //       if (timescaledbVersion) {
//       //         updateDatasourcePluginJsonDataOption({ options, onOptionsChange }, 'timescaledb', true);
//       //       }
//       //     }
//       //     const major = Math.trunc(versionNumber / 100);
//       //     const minor = versionNumber % 100;
//       //     let name = String(major);
//       //     if (versionNumber < 1000) {
//       //       name = String(major) + '.' + String(minor);
//       //     }
//       //     // if (!turboVersions.find((p) => p.value === versionNumber)) {
//       //     //   setVersionOptions((prev) => [...prev, { label: name, value: versionNumber }]);
//       //     // }
//       //     // if (options.jsonData.turboVersion === undefined || options.jsonData.turboVersion !== versionNumber) {
//       //     //   updateDatasourcePluginJsonDataOption({ options, onOptionsChange }, 'turboVersion', versionNumber);
//       //     // }
//       //   }
//       // }
//     };
//     // This logic is only going to run when we create a new datasource
//     if (isValidConfig(options)) {
//       getVersion();
//     }
//   }, [options, saved, setVersionOptions]);
// }

// function isValidConfig(options: DataSourceSettings<TurboOptions, SecureJsonData>) {
//   return (
//     options.url &&
//     options.jsonData.database //&&
//     // options.user &&
//     // (options.secureJsonData?.password || options.secureJsonFields?.password) &&
//     // (options.jsonData.sslmode === TurboTLSModes.disable ||
//     //   (options.jsonData.sslCertFile && options.jsonData.sslKeyFile && options.jsonData.sslRootCertFile)) &&
//     // !options.jsonData.turboVersion &&
//     // !options.readOnly
//   );
// }

// Code generated - EDITING IS FUTILE. DO NOT EDIT.
//
// Generated by:
//     public/app/plugins/gen.go
// Using jennies:
//     TSTypesJenny
//     LatestMajorsOrXJenny
//     PluginEachMajorJenny
//
// Run 'make gen-cue' from repository root to regenerate.

import * as common from '@grafana/schema';

export const pluginVersion = "10.0.2";

export interface Options extends common.SingleStatBaseOptions {
  displayMode: common.BarGaugeDisplayMode;
  minVizHeight: number;
  minVizWidth: number;
  showUnfilled: boolean;
  valueMode: common.BarGaugeValueMode;
}

export const defaultOptions: Partial<Options> = {
  displayMode: common.BarGaugeDisplayMode.Gradient,
  minVizHeight: 10,
  minVizWidth: 0,
  showUnfilled: true,
  valueMode: common.BarGaugeValueMode.Color,
};

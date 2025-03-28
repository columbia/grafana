---
aliases:
  - ../data-sources/prometheus/
  - ../features/datasources/prometheus/
description: Guide for using Prometheus in Grafana
keywords:
  - grafana
  - prometheus
  - guide
menuTitle: Prometheus
title: Prometheus data source
weight: 1300
---

# Prometheus data source

Prometheus is an open-source database that uses an telemetry collector agent to scrape and store metrics used for monitoring and alerting. If you are just getting started with Prometheus, see [What is Prometheus?]({{< relref "../../fundamentals/intro-to-prometheus/" >}}).

Grafana provides native support for Prometheus.
For instructions on downloading Prometheus see [Get started with Grafana and Prometheus](/docs/grafana/latest/getting-started/get-started-grafana-prometheus/#get-started-with-grafana-and-prometheus).

For instructions on how to add a data source to Grafana, refer to the [administration documentation]({{< relref "../../administration/data-source-management/" >}}).
Only users with the organization `administrator` role can add data sources and edit existing data sources.
Administrators can also [configure the data source via YAML]({{< relref "#provision-the-data-source" >}}) with Grafana's provisioning system.

Once you've added the Prometheus data source, you can [configure it](/docs/grafana/latest/datasources/prometheus/configure-prometheus-data-source/) so that your Grafana instance's users can create queries in its [query editor]({{< relref "./query-editor/" >}}) when they [build dashboards]({{< relref "../../dashboards/build-dashboards/" >}}), use [Explore]({{< relref "../../explore/" >}}), and [annotate visualizations]({{< relref "./query-editor/#apply-annotations" >}}).

The following guides will help you get started with the Prometheus data source:

- [Configure the Prometheus data source](/docs/grafana/latest/datasources/prometheus/configure-prometheus-data-source/)
- [Prometheus query editor](/docs/grafana/latest/datasources/prometheus/query-editor/)
- [Template variables](/docs/grafana/latest/datasources/prometheus/template-variables/)

## Prometheus API

The Prometheus data source also works with other projects that implement the [Prometheus querying API](https://prometheus.io/docs/prometheus/latest/querying/api/).

For more information on how to query other Prometheus-compatible projects from Grafana, refer to the specific project's documentation:

- [Grafana Mimir](/docs/mimir/latest/)
- [Thanos](https://thanos.io/tip/components/query.md/)

## Provision the data source

You can define and configure the data source in YAML files as part of Grafana's provisioning system.
For more information about provisioning, and for available configuration options, refer to [Provisioning Grafana]({{< relref "../../administration/provisioning/#data-sources" >}}).

{{% admonition type="note" %}}
Once you have provisioned a data source you cannot edit it.
{{% /admonition %}}

### Provisioning example

```yaml
apiVersion: 1


datasources:
- name: Prometheus
type: prometheus
# Access mode - proxy (server in the UI) or direct (browser in the UI).
access: proxy
url: http://localhost:9090
jsonData:
httpMethod: POST
manageAlerts: true
prometheusType: Prometheus
prometheusVersion: 2.44.0
incrementalQuerying: true
incrementalQueryOverlapWindow: 10m
cacheLevel: 'High'
incrementalQuerying: true
incrementalQueryOverlapWindow: 10m
exemplarTraceIdDestinations:
# Field with internal link pointing to data source in Grafana.
# datasourceUid value can be anything, but it should be unique across all defined data source uids.
- datasourceUid: my_jaeger_uid
name: traceID


# Field with external link.
- name: traceID
url: 'http://localhost:3000/explore?orgId=1&left=%5B%22now-1h%22,%22now%22,%22Jaeger%22,%7B%22query%22:%22$${__value.raw}%22%7D%5D'
```

## View Grafana metrics with Prometheus

Grafana exposes metrics for Prometheus on the `/metrics` endpoint.
We also bundle a dashboard within Grafana so you can start viewing your metrics faster.

**To import the bundled dashboard:**

1. Navigate to the data source's [configuration page]({{< relref "#configure-the-data-source" >}}).
1. Select the **Dashboards** tab.

This displays dashboards for Grafana and Prometheus.

1. Select **Import** for the dashboard to import.

For details about these metrics, refer to [Internal Grafana metrics]({{< relref "../../setup-grafana/set-up-grafana-monitoring/" >}}).

## Amazon Managed Service for Prometheus

The Prometheus data source works with Amazon Managed Service for Prometheus.

If you use an AWS Identity and Access Management (IAM) policy to control access to your Amazon Elasticsearch Service domain, you must use AWS Signature Version 4 (AWS SigV4) to sign all requests to that domain.

For details on AWS SigV4, refer to the [AWS documentation](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html).

### AWS Signature Version 4 authentication

{{% admonition type="note" %}}
Available in Grafana v7.3.5 and higher.
{{% /admonition %}}

To connect the Prometheus data source to Amazon Managed Service for Prometheus using SigV4 authentication, refer to the AWS guide to [Set up Grafana open source or Grafana Enterprise for use with AMP](https://docs.aws.amazon.com/prometheus/latest/userguide/AMP-onboard-query-standalone-grafana.html).

If you run Grafana in an Amazon EKS cluster, follow the AWS guide to [Query using Grafana running in an Amazon EKS cluster](https://docs.aws.amazon.com/prometheus/latest/userguide/AMP-onboard-query-grafana-7.3.html).

## Exemplars

Exemplars associate higher-cardinality metadata from a specific event with traditional time series data. See [Introduction to exemplars]({{< relref "../../fundamentals/exemplars/" >}}) in Prometheus documentation for detailed information on how they work.

{{% admonition type="note" %}}
Available in Prometheus v2.26 and higher with Grafana v7.4 and higher.
{{% /admonition %}}

Grafana 7.4 and higher can show exemplars data alongside a metric both in Explore and in Dashboards.

{{< figure src="/static/img/docs/v74/exemplars.png" class="docs-image--no-shadow" caption="Screenshot showing the detail window of an Exemplar" >}}

See the Exemplars section in [Configure Prometheus data source](/docs/grafana/latest/datasources/prometheus/configure-prometheus-data-source/).

{{< figure src="/static/img/docs/prometheus/exemplars-10-1.png" max-width="500px" class="docs-image--no-shadow" caption="Exemplars" >}}

## Incremental dashboard queries (beta)

As of Grafana 10, the Prometheus data source can be configured to query live dashboards incrementally, instead of re-querying the entire duration on each dashboard refresh.

This can be toggled on or off in the data source configuration or provisioning file (under `incrementalQuerying` in jsonData).
Additionally, the amount of overlap between incremental queries can be configured using the `incrementalQueryOverlapWindow` jsonData field, the default value is `10m` (10 minutes).

Increasing the duration of the `incrementalQueryOverlapWindow` will increase the size of every incremental query, but might be helpful for instances that have inconsistent results for recent data.

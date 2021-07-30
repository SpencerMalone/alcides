# Alcides

Alcides acts as a receiver of webhooks from Alertmanager / Prometheus, that sends API requests to Rundeck to create simple automated remediation.

This repo could use some work (tests, CI, CD, releases, etc.), but the core application is one I have been running with success in a private environment for over 2 years. Good luck!

## Environment Vars to Define

- ALCIDES_TOKEN (This will be used by Alcides to validate requests coming from Alertmanager)
- RUNDECK_URL

- RUNDECK_TOKEN
OR
- RUNDECK_USERNAME
- RUNDECK_PASSWORD

Optional variables:
- RUNDECK_VERSION
- RUNDECK_INSECURE

## Local Dev Setup

1. Create a file called `local.env` in the root of the app which defines the above mentioned environment variables. When productionizing, your orchestration layer will need to manage the environment variables we'll put in this.

2. In a terminal tab, run `script/run-local.sh` which will build and start the sever.


## Alertmanager Configuration

Your alert receiever should look something akin to this:

```
- name: rundeck-job
  webhook_configs:
  - send_resolved: false
    url: "${ALCIDES_URL}"
    http_config:
      basic_auth:
        username: "alcides"
        password: "${ALCIDES_TOKEN}"
```

After the alertmanager portion is setup, you can simply write alerts with an alertroute that passes to Alcides. Things we recommend to think about: 
- Don't send resolved notices, you probably don't want to run a job after the alert is resolved
- Watch out for your repeat_interval in your route

## Prometheus Rules

Your prom alert rules need a required thing, and one optional thing.

- rundeck_job_id is a required label. This is the job ID that will be run when your alert fires
- rundeck_args is an optional rundeck argument string. Read rundeck API docs on how to set these up.

Here's an example rule that includes a prom label from the data in the job argument:

```
  - alert: My thing broke
    expr: up{job="myjob"} == 0
    for: 5m
    labels:
      alertroute: rundeck-job
      rundeck_job_id: c642cad5-f0b7-4b63-91ce-97e96129390d
    annotations:
      rundeck_args: "-HostRegex {{$labels.instance}}"
```


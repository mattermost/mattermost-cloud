# Annotation

## Overview

Both Clusters and Installations can be annotated with a set of Annotations. The Annotations take part during the scheduling of Installations to the Clusters.

## Scheduling with Annotations

Annotations constrain on which Clusters the Installations can be scheduled.

The scheduling is guided by the following principles:
- Installations can be scheduled on any Cluster which contains all Annotations present on the Installation.
- Clusters can contain any additional Annotations not present on the Installation.
- Installation without any Annotations can be scheduled on any available Cluster.

### Examples 

1.
    - Given and Installation: **IA** with Annotations: **private**, **customer-a**.
    - Given Clusters: 
        - **CA** with Annotations: **private**, **customer-a**.
        - **CB** with Annotations: **private**, **customer-b**.
        - **CC** with Annotations: **private**, **customer-a**, **aws-eu**.
    - The Installation can be scheduled on Clusters: **CA** and **CC**.

2.
    - Given and Installation: **IA** with Annotations: **private**, **customer-a**, **aws-us**
    - Given Clusters: 
        - **CA** with Annotations: **private**, **customer-a**.
        - **CB** with Annotations: **private**, **customer-b**, **aws-us**.
        - **CC** with no Annotations.
    - The Installation cannot be scheduled on any Cluster.


## Assigning annotations

Annotations can be assigned to the Cluster and Installation during the creation.

 
### With REST API

In the REST API, Annotations are passed as an array of strings to the JSON payload:
- For Clusters:  
`POST /api/clusters`
    ```json
    {
      ...
      "annotations": ["multi-tenant", "test1"],
      ...
    }
    ```
- For Installations:  
`POST /api/installations`
    ```json
    {
      ...
      "Annotations": ["multi-tenant", "test1"],
      ...
    }
    ```
The name capitalisation differ for the sake of consistency with the rest of the payload. 


### With cloud CLI

To assign annotations via `cloud` CLI, use `--annotation` flag:
```bash 
cloud installation create --owner example --size 100users --affinity multitenant --dns dns.example.com --annotation multi-tenant --annotation test1
```

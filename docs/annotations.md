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
For existing Installations and Clusters, there are dedicated endpoints to manipulate Annotations. 

To preserve Annotations state matching the scheduling state when modifying Annotations on existing resources, the following constraints apply to the operations:
- When deleting Annotation from the Cluster - the Annotation which is being deleted cannot be present on any Installation currently scheduled on that Cluster.
- When adding Annotation to the Installation - the Annotation needs to be present in every Cluster on which the Installation is scheduled.

### With REST API

When creating Cluster or Installation with the REST API, Annotations are passed as an array of strings to the JSON payload:
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
> NOTE: The name capitalisation differ for the sake of consistency with the rest of the payload. 

To modify Annotations on an existing object, the following endpoints can be used:
- To add Annotations to the Cluster:
`POST /api/cluster/[CLUSTER_ID]/annotations`
    ```json
    {
      "annotations": ["multi-tenant", "test1"]
    } 
    ```
- To remove an Annotation from the Cluster:  
`DELETE /api/cluster/[CLUSTER_ID]/annotation/[ANNOTATION_NAME]`
- To add Annotations to the Installation:
`POST /api/installation/[INSTALLATION_ID]/annotations`
    ```json
    {
      "annotations": ["multi-tenant", "test1"]
    } 
    ```
- To remove an Annotation from the Installation:  
`DELETE /api/installation/[INSTALLATION_ID]/annotation/[ANNOTATION_NAME]`


### With cloud CLI

To assign annotations via `cloud` CLI, use `--annotation` flag:
```bash 
cloud installation create --owner example --size 100users --affinity multitenant --dns dns.example.com --annotation multi-tenant --annotation test1
```

To add Annotations to the existing Cluster, use:
```bash
cloud cluster annotation add --cluster [CLUSTER_ID] --annotation multi-tenant --annotation test1
``` 
To remove, use:
```bash
cloud cluster annotation delete --cluster [CLUSTER_ID] --annotation multi-tenant
``` 
> NOTE: You can only delete a single Annotation at a time.

For Installations, use `installation` instead of `cluster`.

```mermaid
---
title: "Test Workflow"
---
flowchart TB
    build["build"]
    test["test"]
    deploy["deploy"]
    build --> test
    test --> deploy
```

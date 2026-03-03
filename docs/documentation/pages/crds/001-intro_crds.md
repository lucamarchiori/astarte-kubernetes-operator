# Introduction

The Astarte Operator extends the Kubernetes API through the definition of Custom Resources.

To browse the CRD documentation, [follow this link](./crds/index.html).

## Custom Annotations
Astarte and ADI CRs support a set of custom annotations that can be used to toggle custom behaviors that are not directly supported by the CRD schema. This is often the case for features that are still experimental, or that are not expected to be widely used, and that would therefore add unnecessary complexity to the CRD schema.

### Astarte CR

**Enable or disable the Astarte Dashboard sidebar**
- Annotation: `api.astarte-platform.org/hide-dashboard-sidebar`
- Values: `"true"` or `"false"`

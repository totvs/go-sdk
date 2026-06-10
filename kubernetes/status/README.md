# Kubernetes Status (`kubernetes/status`)

Módulo para padronização de status em recursos Kubernetes. Encapsula [KStatus](https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus) oficial e expõe helpers de escrita/leitura seguindo as convenções da spec KStatus.

---

## Conceitos

O módulo cobre dois lados da spec KStatus:

**Escrita (operator/controller)** — convenção de como persistir condições no CRD:
- `Ready`, `Reconciling`, `Stalled` como `type` corretos
- `observedGeneration` consistente com `metadata.generation`
- `type` único por condition (upsert, não append)

**Leitura (qualquer consumidor)** — computação on-the-fly a partir dos sinais do objeto:

```text
observedGeneration != generation  → InProgress  (stale)
Reconciling=True                  → InProgress
Stalled=True                      → Failed
Ready=True                        → Current
Ready=False/Unknown               → InProgress
deletionTimestamp presente        → Terminating
```

> `Summary` é computado na leitura e nunca persistido no CRD.

---

## Setup Rápido

```go
import status "github.com/totvs/go-sdk/kubernetes/status"
```

**Operator escreve no final de cada reconcile:**

```go
// sucesso
status.MarkReady(&conditions, observedGeneration, status.Reasons.Reconciled, "reconciled")

// em progresso
status.MarkReconciling(&conditions, observedGeneration, status.Reasons.Reconciling, "installing...")

// bloqueio/erro terminal
status.MarkStalled(&conditions, observedGeneration, status.Reasons.DependencyNotFound, "CRD not found")

// aguardando input externo
status.MarkWaiting(&conditions, observedGeneration, "PendingApproval", "awaiting approval")

// deleção
status.MarkTerminating(&conditions, observedGeneration, "terminating")
```

**Consumidor lê:**

```go
summary, err := status.SummaryFromObject(obj)
// summary.KStatus  → "Failed"
// summary.State    → "error"
// summary.Severity → "error"
// summary.Reason   → "DependencyNotFound"
// summary.Message  → "CRD not found"
```

---

## Estrutura

| Arquivo | Responsabilidade |
|---|---|
| `types.go` | `KStatus`, `State`, `Severity`, `Reason`, `Reasons`, `Summary` |
| `conditions.go` | `Mark*`, `NewCondition`, `SetCondition`, `FindCondition`, `IsCondition*` |
| `summary.go` | `SummaryFromObject`, `SummaryFromUnstructured`, `NotFoundSummary`, `WithSummaryMapping` |
| `summary_mapping.go` | `SummaryMapping`, `SummaryRule`, `DefaultSummaryMapping`, `Merge`, `Lookup` |
| `compute.go` | `Compute`, `ComputeFromUnstructured` — KStatus puro sem Summary |
| `unstructured.go` | helpers internos para ler conditions/generation de `Unstructured` |

---

## Write Helpers

Todos os `Mark*` garantem:
- `Ready` / `Reconciling` / `Stalled` como `type` correto
- `observedGeneration` preenchido
- unicidade de `type` (upsert via `meta.SetStatusCondition`)
- remoção dos sinais conflitantes (ex: `MarkReady` remove `Reconciling` e `Stalled`)

| Helper | Ready | Reconciling | Stalled | KStatus resultante |
|---|---|---|---|---|
| `MarkReady` | True | removido | removido | Current |
| `MarkReconciling` | False | True | removido | InProgress |
| `MarkWaiting` | False | removido | removido | InProgress |
| `MarkStalled` | False | removido | True | Failed |
| `MarkTerminating` | False | True | removido | Terminating* |

> *`MarkTerminating` seta as conditions. KStatus lê `deletionTimestamp` do objeto real para classificar como `Terminating`.

### Quando usar `MarkStalled`

Use para qualquer bloqueio terminal — quando o controller não pode avançar sem intervenção:

```go
// dependência ausente ou indisponível
status.MarkStalled(&conditions, gen, status.Reasons.DependencyNotFound, "secret not found")
status.MarkStalled(&conditions, gen, status.Reasons.DependencyUnavailable, "database not ready")

// spec inválida
status.MarkStalled(&conditions, gen, status.Reasons.InvalidConfiguration, "invalid helm values")

// permissão negada
status.MarkStalled(&conditions, gen, status.Reasons.PermissionDenied, "SA missing cluster-admin")

// timeout de operação externa
status.MarkStalled(&conditions, gen, status.Reasons.Timeout, "webhook timed out")
```

### Quando usar `MarkWaiting`

Use para estados de espera legítimos — não são falhas, o controller aguarda intencionalmente:

```go
// aguardando input humano
status.MarkWaiting(&conditions, gen, "PendingApproval", "awaiting ops approval")

// aguardando pré-condição (TLS, quota, etc)
status.MarkWaiting(&conditions, gen, status.Reasons.PreconditionNotMet, "TLS cert not issued")
```

---

## Reasons

`Reasons` é o vocabulário de reasons embutido no SDK. Acesse via `status.Reasons.<Tab>` para autocomplete.

### Core (usados internamente pelos `Mark*`)

| Reason | Analogia HTTP | Usado em |
|---|---|---|
| `Reasons.Reconciled` | ~200 OK | `MarkReady` |
| `Reasons.Reconciling` | ~102 Processing | `MarkReconciling` |
| `Reasons.Terminating` | ~102 Processing | `MarkTerminating` |
| `Reasons.Unknown` | ~520 Unknown | `NotFoundSummary` |

### Common (vocabulário sugerido para controllers)

| Reason | Analogia HTTP | Estado mapeado |
|---|---|---|
| `Reasons.InvalidConfiguration` | ~400 Bad Request | `error / error` |
| `Reasons.DependencyNotFound` | ~404 Not Found | `error / error` |
| `Reasons.DependencyUnavailable` | ~503 Service Unavailable | `error / warning` |
| `Reasons.Conflict` | ~409 Conflict | `error / warning` |
| `Reasons.PreconditionNotMet` | ~428 Precondition Required | `waiting / warning` |
| `Reasons.PermissionDenied` | ~403 Forbidden | `error / error` |
| `Reasons.Timeout` | ~408 Request Timeout | `error / warning` |

Reasons de domínio específico (ex: `PendingApproval`, `LicenseExpired`) pertencem ao operator/consumer — não ao SDK.

---

## Summary

`Summary` é o contrato de leitura para API e frontend. Computado on-the-fly, nunca persistido no CRD.

```go
type Summary struct {
    KStatus  KStatus  // estado técnico de reconciliação — interop com tooling k8s
    State    State    // estado de produto — campo principal para UI/API
    Severity Severity // severidade para renderização visual
    Reason   Reason   // chave estável de mapeamento (omitempty)
    Message  string   // mensagem humana/técnica (omitempty)
}
```

### KStatus — valores possíveis

| KStatus | Quando |
|---|---|
| `Current` | `Ready=True` |
| `InProgress` | `Reconciling=True` ou `observedGeneration` desatualizado |
| `Failed` | `Stalled=True` |
| `Terminating` | `deletionTimestamp` presente |
| `NotFound` | recurso ausente no cluster |
| `Unknown` | não determinável |

### State — valores possíveis

`ready` · `progressing` · `waiting` · `warning` · `error` · `terminating` · `notFound` · `unknown`

### Severity — valores possíveis

`success` · `info` · `warning` · `error`

---

## SummaryMapping

`SummaryMapping` define como cada `Reason` é mapeado para `State` + `Severity`.

```go
type SummaryMapping map[Reason]SummaryRule

type SummaryRule struct {
    State    State
    Severity Severity
}
```

`DefaultSummaryMapping()` cobre os 4 core reasons + 7 common reasons. Se a `Reason` não estiver no mapeamento, o SDK usa os defaults por KStatus:

```text
Current     → state:ready,       severity:success
InProgress  → state:progressing, severity:info
Failed      → state:error,       severity:error
Terminating → state:terminating, severity:warning
NotFound    → state:notFound,    severity:warning
Unknown     → state:unknown,     severity:warning
```

### Customizando com reasons de domínio

```go
// definido no operator/pkg/platformstatus (ou equivalente)
const ReasonPendingApproval status.Reason = "PendingApproval"

domainMapping := status.SummaryMapping{
    ReasonPendingApproval: {State: status.StateWaiting, Severity: status.SeverityWarning},
}

// WithSummaryMapping faz merge sobre DefaultSummaryMapping — base técnica intacta
summary, err := status.SummaryFromObject(obj, status.WithSummaryMapping(domainMapping))
```

---

## Conditions de Domínio

Use `NewCondition` + `SetCondition` para conditions de negócio que **complementam** (não substituem) `Ready`/`Reconciling`/`Stalled`:

```go
const ConditionResourceApproved = "ResourceApproved"

// operator escreve — KStatus signal + condition de domínio
status.MarkWaiting(&conditions, gen, "PendingApproval", "awaiting approval")

domainCond := status.NewCondition(
    ConditionResourceApproved,
    metav1.ConditionFalse,
    "PendingApproval",
    "not yet approved",
    gen,
)
status.SetCondition(&conditions, domainCond)

// consumer lê — sem string literal
approved := status.IsConditionTrue(conditions, ConditionResourceApproved)
c := status.FindCondition(conditions, ConditionResourceApproved)
```

---

## Compute — KStatus puro

Use quando precisar apenas do estado técnico, sem `Summary`:

```go
// a partir de qualquer client.Object
ks, err := status.Compute(obj)

// a partir de Unstructured (dynamic client, addon-agent)
ks, err := status.ComputeFromUnstructured(u)

// ks é um dos: KStatusCurrent, KStatusInProgress, KStatusFailed,
//              KStatusTerminating, KStatusNotFound, KStatusUnknown
```

---

## Arquitetura de Responsabilidades

```
go-sdk/kubernetes/status
  └── DefaultSummaryMapping()    ← base técnica genérica (core + common)
  └── Reasons.*                  ← vocabulário genérico

operator/pkg/platformstatus      ← domínio específico do operator
  ├── conditions.go              ← constantes de ConditionType de domínio
  └── mappings.go                ← SummaryMappings por recurso

service-core / addon-agent
  └── SummaryFromObject(obj, WithSummaryMapping(platformstatus.ResourceMapping()))
```

Mudança em mapeamento ou constante de domínio: **um lugar, propaga para todos os consumers**.

---

## Exemplo Completo

```go
package main

import (
    "log"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    status "github.com/totvs/go-sdk/kubernetes/status"
)

func reconcile(obj *MyResource, conditions *[]metav1.Condition) {
    gen := obj.Generation

    // dependency not found → stalled
    if !dependencyExists() {
        status.MarkStalled(conditions, gen, status.Reasons.DependencyNotFound, "config-secret not found")
        return
    }

    // installing
    status.MarkReconciling(conditions, gen, status.Reasons.Reconciling, "applying helm chart")
    if err := applyHelmChart(); err != nil {
        status.MarkStalled(conditions, gen, status.Reasons.InvalidConfiguration, err.Error())
        return
    }

    // done
    status.MarkReady(conditions, gen, status.Reasons.Reconciled, "helm chart ready")
}

func readStatus(obj *MyResource) {
    // consumer com mapeamento de domínio
    domainMapping := status.SummaryMapping{
        "PendingApproval": {State: status.StateWaiting, Severity: status.SeverityWarning},
    }

    summary, err := status.SummaryFromObject(obj, status.WithSummaryMapping(domainMapping))
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("state=%s severity=%s reason=%s", summary.State, summary.Severity, summary.Reason)
}
```

Exemplo executável completo: [`examples/kubernetes-status/main.go`](../../examples/kubernetes-status/main.go)

```bash
make example-run example=kubernetes-status
```

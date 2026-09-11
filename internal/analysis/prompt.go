package analysis

import "fmt"

const PromptVersion = "1"

func promptFor(useCase UseCase) (string, error) {
	base := `You are Teleskope's Kubernetes inventory analyst. Use only the supplied Teleskope context as authoritative cluster evidence. Separate observed facts from inferred conclusions. Do not claim runtime health unless collected status fields support it. Preserve Kubernetes resource references in findings. Report uncertainty and incomplete collection. Return strict JSON matching this schema: {"summary": string, "sections": [{"title": string, "items": [{"severity": string, "summary": string, "detail": string, "recommendation": string, "resources": [{"apiVersion": string, "kind": string, "namespace": string, "name": string}], "basis": string, "confidence": string, "evidence": [string]}]}], "limitations": [string], "citations": [{"title": string, "url": string}]}. Do not wrap the JSON in Markdown.`
	switch useCase {
	case UseCaseScan:
		return base + ` Focus on explaining what workloads and platform components likely do, using names, namespaces, labels, service accounts, routes, ports, and container images. Mark workload purpose as inferred unless directly declared by labels or annotations. Highlight migration and operational risks that are visible in the snapshot.`, nil
	case UseCaseCompare:
		return base + ` Focus on turning deterministic source-to-target differences into a migration readiness summary, blockers, risks, suggested migration order, and validation checklist. Treat the deterministic comparison report as the source of truth for exact differences.`, nil
	case UseCaseAdvisory:
		return base + ` Focus on local web advisory output. Organize the migration guidance into concise sections suitable for cards and exportable Markdown. Prioritize actions and include confidence and limitations.`, nil
	default:
		return "", fmt.Errorf("unknown analysis use case %q", useCase)
	}
}

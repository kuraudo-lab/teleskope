package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func WriteJSON(w io.Writer, result Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func WriteHuman(w io.Writer, result Result) error {
	fmt.Fprintf(w, "teleskope llm analysis\n")
	fmt.Fprintf(w, "use case   %s\n", value(string(result.UseCase)))
	fmt.Fprintf(w, "model      %s\n", value(result.Model))
	fmt.Fprintf(w, "summary    %s\n\n", value(result.Summary))
	for _, section := range result.Sections {
		fmt.Fprintf(w, "%s\n", section.Title)
		for _, item := range section.Items {
			fmt.Fprintf(w, "  %s %s\n", severityMarker(item.Severity), item.Summary)
			if item.Detail != "" {
				fmt.Fprintf(w, "    %s\n", item.Detail)
			}
			if item.Recommendation != "" {
				fmt.Fprintf(w, "    next: %s\n", item.Recommendation)
			}
		}
	}
	if len(result.Limitations) > 0 {
		fmt.Fprintf(w, "\nlimitations\n")
		for _, limitation := range result.Limitations {
			fmt.Fprintf(w, "  - %s\n", limitation)
		}
	}
	return nil
}

func Markdown(result Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Teleskope LLM analysis\n\n")
	fmt.Fprintf(&b, "%s\n\n", result.Summary)
	fmt.Fprintf(&b, "| Field | Value |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Use case | %s |\n", mdCell(string(result.UseCase)))
	fmt.Fprintf(&b, "| Provider | %s |\n", mdCell(result.Provider))
	fmt.Fprintf(&b, "| Model | %s |\n", mdCell(result.Model))
	fmt.Fprintf(&b, "| Prompt version | %s |\n", mdCell(result.PromptVersion))
	fmt.Fprintf(&b, "| Web search | %t |\n\n", result.WebSearch)
	for _, section := range result.Sections {
		fmt.Fprintf(&b, "## %s\n\n", mdInline(section.Title))
		if len(section.Items) == 0 {
			fmt.Fprintf(&b, "No items.\n\n")
			continue
		}
		for _, item := range section.Items {
			label := strings.TrimSpace(strings.Join(nonEmpty(item.Severity, item.Basis, item.Confidence), "; "))
			if label != "" {
				fmt.Fprintf(&b, "- **%s** (%s)\n", mdInline(item.Summary), mdInline(label))
			} else {
				fmt.Fprintf(&b, "- **%s**\n", mdInline(item.Summary))
			}
			if item.Detail != "" {
				fmt.Fprintf(&b, "  - Detail: %s\n", mdInline(item.Detail))
			}
			if item.Recommendation != "" {
				fmt.Fprintf(&b, "  - Recommendation: %s\n", mdInline(item.Recommendation))
			}
			if len(item.Resources) > 0 {
				refs := make([]string, 0, len(item.Resources))
				for _, ref := range item.Resources {
					refs = append(refs, refValue(ref.Kind, ref.Namespace, ref.Name))
				}
				fmt.Fprintf(&b, "  - Resources: %s\n", mdInline(strings.Join(refs, ", ")))
			}
			if len(item.Evidence) > 0 {
				fmt.Fprintf(&b, "  - Evidence: %s\n", mdInline(strings.Join(item.Evidence, ", ")))
			}
		}
		fmt.Fprintln(&b)
	}
	if len(result.Limitations) > 0 {
		fmt.Fprintf(&b, "## Limitations\n\n")
		for _, limitation := range result.Limitations {
			fmt.Fprintf(&b, "- %s\n", mdInline(limitation))
		}
		fmt.Fprintln(&b)
	}
	if len(result.Citations) > 0 {
		fmt.Fprintf(&b, "## Citations\n\n")
		for _, citation := range result.Citations {
			if citation.URL != "" {
				fmt.Fprintf(&b, "- [%s](%s)\n", mdInline(firstNonEmpty(citation.Title, citation.URL)), citation.URL)
			} else {
				fmt.Fprintf(&b, "- %s\n", mdInline(citation.Title))
			}
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func severityMarker(severity string) string {
	switch strings.ToLower(severity) {
	case "blocker", "critical", "error":
		return "✗"
	case "warning", "warn", "risk":
		return "!"
	case "info", "low":
		return "•"
	default:
		return "-"
	}
}

func value(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func mdCell(value string) string {
	value = mdInline(value)
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

func mdInline(value string) string {
	value = strings.ReplaceAll(value, "`", "\\`")
	return value
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func refValue(kind, namespace, name string) string {
	if namespace != "" {
		return kind + "/" + namespace + "/" + name
	}
	if kind != "" {
		return kind + "/" + name
	}
	return name
}

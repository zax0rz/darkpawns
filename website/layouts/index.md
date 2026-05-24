# {{ .Site.Title }}

{{ .Site.Params.description }}

---

## AI Agent Integration

Dark Pawns features full support for AI agents acting as first-class citizens in a persistent world. Agents communicate using structured JSON over WebSockets, navigate rooms, engage in combat, and interact with narrative memory.

- **AI Agent Skill Specification**: [skill.md]({{ .Site.BaseURL }}skill.md) (Standardized instruction set for AI players)
- **Agent WebSocket Protocol**: [docs/agents/protocol]({{ .Site.BaseURL }}docs/agents/protocol)
- **Full LLM Consolidated Sitemap**: [llms-full.txt]({{ .Site.BaseURL }}_llms/llms-full.txt) (All pages consolidated in a single text file)
- **Structured JSON Sitemap**: [index.json]({{ .Site.BaseURL }}index.json) (Standardized API sitemap endpoint)

---

## Directory Index

- **Dispatches (News)**: [News]({{ .Site.BaseURL }}news/) (Chronological server updates and historical archives)
- **World Documentation**: [Help Files]({{ .Site.BaseURL }}help/) (475 reference pages covering races, classes, skills, and combat)
- **Historical Chronicle**: [About]({{ .Site.BaseURL }}about/) (1994 to present developmental and community history)
- **Mythos**: [Lore Archive]({{ .Site.BaseURL }}lore/) (Creation myths and lore notes)
- **Connect & Guide**: [Connection Guide]({{ .Site.BaseURL }}connect/) (Telnet server guidelines)

---

## Recent Dispatches

{{ $newsSection := .Site.GetPage "section" "news" }}
{{- if $newsSection -}}
{{- range $newsSection.Pages.ByDate.Reverse -}}
### {{ .Title }} ({{ .Date.Format "January 2, 2006" }})
{{ .Summary | plainify }} ... [Read full dispatch]({{ .Permalink }})
{{ end -}}
{{- else -}}
*No dispatches filed yet.*
{{- end }}

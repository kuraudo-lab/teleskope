# Stable topology type colors

User acceptance of PR #62 found that object colors change across drilldown levels. Object type must retain its semantic color within each theme across overview, selection, relationships, hover and keyboard focus. Indicate interaction using shape emphasis/width and glow, without replacing the type hue. Preserve health badges and graph algorithms.

Reproduce against the real embedded renderer using computed SVG colors before and after navigation/selection. Add browser regression coverage for both themes, regenerate embedded assets/demo reports, run relevant browser and Go checks, and update the existing PR and local acceptance server.

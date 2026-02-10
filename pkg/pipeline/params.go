package pipeline

import (
	"fmt"
	"strings"
)

// ValidateParams checks that all required params are provided and no unknown
// params are passed.
func ValidateParams(defs []ParamDef, provided map[string]string) error {
	known := make(map[string]struct{}, len(defs))
	for i := range defs {
		name := defs[i].Name
		if _, ok := known[name]; ok {
			return fmt.Errorf("param %q: %w", name, ErrDuplicateParam)
		}
		known[name] = struct{}{}
		if defs[i].Required {
			if _, ok := provided[name]; !ok {
				return fmt.Errorf("param %q: %w", name, ErrMissingParam)
			}
		}
	}
	for k := range provided {
		if _, ok := known[k]; !ok {
			return fmt.Errorf("param %q: %w", k, ErrUnknownParam)
		}
	}
	return nil
}

// ResolveParams merges provided values with defaults, returning the final
// param map. Calls ValidateParams internally.
func ResolveParams(defs []ParamDef, provided map[string]string) (map[string]string, error) {
	if err := ValidateParams(defs, provided); err != nil {
		return nil, err
	}
	resolved := make(map[string]string, len(defs))
	for i := range defs {
		if v, ok := provided[defs[i].Name]; ok {
			resolved[defs[i].Name] = v
		} else {
			resolved[defs[i].Name] = defs[i].Default
		}
	}
	return resolved, nil
}

// SubstituteParams replaces ${param.*} placeholders in all step fields.
// Uses strings.NewReplacer for single-pass substitution, matching the
// approach used by substituteVars for ${matrix.*}.
func SubstituteParams(steps []Step, params map[string]string) []Step {
	if len(params) == 0 {
		return steps
	}
	combo := make(map[string]string, len(params))
	for k, v := range params {
		combo["param."+k] = v
	}
	return mapSlice(steps, func(s Step) Step {
		return substituteStepVars(s, combo)
	})
}

// substituteStepVars applies ${prefix.key} substitution to all string fields
// of a step. This reuses the same pattern as replicateStep but with an
// arbitrary namespace prefix baked into the combo keys.
func substituteStepVars(s Step, combo map[string]string) Step {
	sub := func(v string) string { return substituteNamespacedVars(v, combo) }
	return Step{
		Name:      s.Name,
		Image:     sub(s.Image),
		Workdir:   sub(s.Workdir),
		Platform:  sub(s.Platform),
		Matrix:    s.Matrix,
		Run:       mapSlice(s.Run, sub),
		DependsOn: mapSlice(s.DependsOn, func(d string) string { return d }),
		Mounts: mapSlice(s.Mounts, func(m Mount) Mount {
			return Mount{Source: sub(m.Source), Target: sub(m.Target), ReadOnly: m.ReadOnly}
		}),
		Caches: mapSlice(s.Caches, func(c Cache) Cache {
			return Cache{ID: sub(c.ID), Target: sub(c.Target)}
		}),
	}
}

// substituteNamespacedVars replaces ${key} placeholders where keys already
// include their namespace prefix (e.g. "param.go-version").
func substituteNamespacedVars(s string, combo map[string]string) string {
	if len(combo) == 0 {
		return s
	}
	pairs := make([]string, 0, len(combo)*2)
	for k, v := range combo {
		pairs = append(pairs, "${"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

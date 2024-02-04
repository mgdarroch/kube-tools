package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	rootCmd.AddCommand(aliasesCmd)
}

var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "generates aliases for kubectl",
	Long:  "generates shorthand aliases for kubectl, e.g. 'kubectl get pods' becomes 'kgpo'.  Heavily inspired by https://github.com/ahmetb/kubectl-aliases",
	Run: func(cmd *cobra.Command, args []string) {
		main()
	},
}

// Part represents a single part of a command (e.g., an operation or a resource)
type Part struct {
	Alias            string
	Full             string
	AllowWhenOneOf   []string
	IncompatibleWith []string
}

// AliasGenerator holds the configuration for generating aliases
type AliasGenerator struct {
	Commands  []Part
	GlobalOps []Part
	Ops       []Part
	Resources []Part
	Args      []Part
	PosArgs   []Part
}

// generate generates and prints all valid aliases based on the generator's configuration
func (ag *AliasGenerator) generate() {
	for _, cmd := range ag.Commands {
		ag.combine([]Part{cmd}, ag.GlobalOps, 1)
	}
}

// combine recursively combines parts and checks their validity
func (ag *AliasGenerator) combine(current []Part, next []Part, depth int) {
	if depth == 6 { // Reached the end of the combination chain
		ag.printAlias(current)
		return
	}

	for _, part := range next {
		if ag.isValidCombination(current, part) {
			ag.nextStep(append(current, part), depth+1)
		}
	}

	// Try without adding a new part from the current group
	ag.nextStep(current, depth+1)
}

// nextStep decides which group of parts to combine next based on the current depth
func (ag *AliasGenerator) nextStep(current []Part, depth int) {
	switch depth {
	case 2:
		ag.combine(current, ag.Ops, depth)
	case 3:
		ag.combine(current, ag.Resources, depth)
	case 4:
		ag.combine(current, ag.Args, depth)
	case 5:
		ag.combine(current, ag.PosArgs, depth)
	default:
		ag.combine(current, []Part{}, depth)
	}
}

// isValidCombination checks if adding a new part to the current combination is valid
func (ag *AliasGenerator) isValidCombination(current []Part, newPart Part) bool {
	currentAliases := make(map[string]struct{})
	for _, part := range current {
		currentAliases[part.Alias] = struct{}{}
		for _, incompatible := range part.IncompatibleWith {
			if incompatible == newPart.Alias {
				return false
			}
		}
	}

	if len(newPart.AllowWhenOneOf) > 0 {
		found := false
		for _, allowed := range newPart.AllowWhenOneOf {
			if _, exists := currentAliases[allowed]; exists {
				found = true
				break
			}
		}
		return found
	}

	return true
}

// printAlias prints the alias for the current combination
func (ag *AliasGenerator) printAlias(combination []Part) {
	alias := ""
	full := ""
	for _, part := range combination {
		alias += part.Alias
		full += part.Full + " "
	}
	fmt.Printf("alias %s='%s'\n", alias, strings.TrimSpace(full))
}

func main() {
	ag := AliasGenerator{
		Commands: []Part{
			{"k", "kubectl", nil, nil},
		},
		GlobalOps: []Part{
			{"sys", "--namespace=kube-system", nil, nil},
		},
		Ops:       generateOperations(),
		Resources: generateResources(),
		Args:      generateArguments(),
		PosArgs:   generatePositionalArgs(generateResourceTypes(generateResources())),
	}

	if len(os.Args) > 1 && (os.Args[1] == "bash" || os.Args[1] == "zsh" || os.Args[1] == "fish") {
		fmt.Printf("# Generated aliases for %s\n", os.Args[1])
	}

	ag.generate()
}

func generateOperations() []Part {
	return []Part{
		{"a", "apply --recursive -f", nil, nil},
		{"ak", "apply -k", nil, []string{"sys"}},
		{"k", "kustomize", nil, []string{"sys"}},
		{"ex", "exec -i -t", nil, nil},
		{"lo", "logs -f", nil, nil},
		{"lop", "logs -f -p", nil, nil},
		{"p", "proxy", nil, []string{"sys"}},
		{"pf", "port-forward", nil, []string{"sys"}},
		{"g", "get", nil, nil},
		{"d", "describe", nil, []string{"sys"}},
		{"rm", "delete", nil, []string{"sys"}},
		{"run", "run --rm --restart=Never --image-pull-policy=IfNotPresent -i -t", nil, nil},
	}
}

func generateResources() []Part {
	return []Part{
		// base k8s
		{"po", "pods", []string{"g", "d", "rm"}, nil},
		{"dep", "deployment", []string{"g", "d", "rm"}, nil},
		{"sts", "statefulset", []string{"g", "d", "rm"}, nil},
		{"svc", "service", []string{"g", "d", "rm"}, nil},
		{"ing", "ingress", []string{"g", "d", "rm"}, nil},
		{"cm", "configmap", []string{"g", "d", "rm"}, nil},
		{"sec", "secret", []string{"g", "d", "rm"}, nil},
		{"no", "nodes", []string{"g", "d"}, []string{"sys"}},
		{"ns", "namespaces", []string{"g", "d"}, []string{"sys"}},
		// istio
		{"vs", "virtualservices", []string{"g", "d", "rm"}, nil},
	}
}

func generateResourceTypes(resources []Part) []string {
	var resourceTypes []string
	for _, resource := range resources {
		resourceTypes = append(resourceTypes, resource.Alias)
	}
	return resourceTypes
}

func generateArguments() []Part {
	return []Part{
		{"oyaml", "-o=yaml", []string{"g"}, []string{"owide", "ojson", "sl"}},
		{"owide", "-o=wide", []string{"g"}, []string{"oyaml", "ojson"}},
		{"ojson", "-o=json", []string{"g"}, []string{"owide", "oyaml", "sl"}},
		{"all", "--all-namespaces", []string{"g", "d"}, []string{"rm", "f", "no", "sys"}},
		{"sl", "--show-labels", []string{"g"}, []string{"oyaml", "ojson"}},
		{"all", "--all", []string{"rm"}, nil},
		{"w", "--watch", []string{"g"}, []string{"oyaml", "ojson", "owide"}},
	}
}

func generatePositionalArgs(resourceTypes []string) []Part {
	resourceTypes = append(resourceTypes, []string{"all", "l", "sys"}...)
	return []Part{
		{"f", "--recursive -f", []string{"g", "d", "rm"}, resourceTypes},
		{"l", "-l", []string{"g", "d", "rm"}, []string{"f", "all"}},
		{"n", "--namespace", []string{"g", "d", "rm", "lo", "ex", "pf"}, []string{"ns", "no", "sys", "all"}},
	}
}

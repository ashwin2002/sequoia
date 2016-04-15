package sequoia

/* Template.go
 *
 * Template Resolver methods
 */

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"text/template"
)

type TemplateResolver struct {
	Scope *Scope
}

func ParseTemplate(s *Scope, command string) string {

	tResolv := TemplateResolver{s}

	netFunc := template.FuncMap{
		"net":    tResolv.Address,
		"bucket": tResolv.BucketName,
	}
	tmpl, err := template.New("t").Funcs(netFunc).Parse(command)
	chkerr(err)

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, &tResolv)
	chkerr(err)

	return fmt.Sprintf("%s", out)
}

// apply scope scale factor to the value
func (t *TemplateResolver) Scale(val int) string {
	scale := t.Scope.TestConfig.Options.Scale
	if scale == 0 {
		scale++
	}
	return strconv.Itoa(val * scale)
}

// resolve nodes with specified service
// .Nodes | .Service `n1ql` | net 0
func (t *TemplateResolver) Service(service string, servers []ServerSpec) []ServerSpec {

	serviceNodes := []ServerSpec{}
	matchIdx := 0
	for _, spec := range servers {
		added := false
		for _, name := range spec.Names {
			rest := t.Scope.Provider.GetRestUrl(name)
			ok := NodeHasService(service, rest, spec.RestUsername, spec.RestPassword)
			if ok == true {
				if added == false {
					serviceNodes = append(serviceNodes, ServerSpec{Names: []string{name}})
					added = true
				} else {
					serviceNodes[matchIdx].Names = append(serviceNodes[matchIdx].Names, name)
				}
			}
		}
		if added == true {
			matchIdx++
		}
	}

	return serviceNodes
}

func (t *TemplateResolver) Nodes() []ServerSpec {
	return t.Scope.Spec.Servers
}

func (t *TemplateResolver) Cluster(index int, servers []ServerSpec) []ServerSpec {
	return []ServerSpec{servers[index]}
}

// Shortcut: .Nodes | .Cluster 0
func (t *TemplateResolver) ClusterNodes() []ServerSpec {
	return t.Cluster(0, t.Nodes())
}

// Shortcut: .ClusterNodes | net 0
func (t *TemplateResolver) Orchestrator() string {
	nodes := t.ClusterNodes()
	name := nodes[0].Names[0]
	val := t.Scope.Provider.GetHostAddress(name)
	return val
}

// Shortcut: .ClusterNodes | .Service `n1ql` | net 0
func (t *TemplateResolver) QueryNode() string {
	nodes := t.ClusterNodes()
	serviceNodes := t.Service("n1ql", nodes)
	return t.Address(0, serviceNodes)
}

// Shortcut: .ClusterNodes | .Service `n1ql` | net N
func (t *TemplateResolver) NthQueryNode(n int) string {
	nodes := t.ClusterNodes()
	serviceNodes := t.Service("n1ql", nodes)
	return t.Address(n, serviceNodes)
}

// Shortcut: .ClusterNodes | .Service `kv` | net 0
func (t *TemplateResolver) DataNode() string {
	nodes := t.ClusterNodes()
	serviceNodes := t.Service("kv", nodes)
	return t.Address(0, serviceNodes)
}

// Shortcut: .ClusterNodes | .Service `kv` | net N
func (t *TemplateResolver) NthDataNode(n int) string {
	nodes := t.ClusterNodes()
	serviceNodes := t.Service("kv", nodes)
	return t.Address(n, serviceNodes)
}

func (t *TemplateResolver) Attr(key string, servers []ServerSpec) string {
	attr := t.Scope.Spec.ToAttr(key)
	spec := reflect.ValueOf(servers[0])
	val := spec.FieldByName(attr).String()
	return val
}

// Shortcut:  .ClusterNodes | .Attr `rest_username`
func (t *TemplateResolver) RestUsername() string {
	nodes := t.ClusterNodes()
	return t.Attr("rest_username", nodes)
}

// Shortcut:  .ClusterNodes | .Attr `rest_password`
func (t *TemplateResolver) RestPassword() string {
	nodes := t.ClusterNodes()
	return t.Attr("rest_password", nodes)
}

// Template function: `net`
func (t *TemplateResolver) Address(index int, servers []ServerSpec) string {
	if len(servers[0].Names) <= index {
		return "<node_not_found>"
	}

	var name = servers[0].Names[index]
	return t.Scope.Provider.GetHostAddress(name)
}

// Template function: `bucket`
func (t *TemplateResolver) BucketName(index int, servers []ServerSpec) string {
	var i = 0
	for _, spec := range servers {
		for _, bucketSpec := range spec.BucketSpecs {
			for _, name := range bucketSpec.Names {
				if i == index {
					return name
				}
			}
		}
	}
	return "<bucket_not_found>"
}

// .ClusterNodes | bucket 0
func (t *TemplateResolver) Bucket() string {
	return t.BucketName(0, t.ClusterNodes())
}

// .ClusterNodes | bucket N
func (t *TemplateResolver) NthBucket(n int) string {
	return t.BucketName(n, t.ClusterNodes())
}
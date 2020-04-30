package introspection

import (
	"sort"

	"github.com/chirino/graphql/internal/lexer"
	"github.com/chirino/graphql/schema"
)

type Schema struct {
	schema *schema.Schema
}

// WrapSchema is only used internally.
func WrapSchema(schema *schema.Schema) *Schema {
	return &Schema{schema}
}

func (r *Schema) Types() []*Type {
	var names []string
	for name := range r.schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	l := make([]*Type, len(names))
	for i, name := range names {
		l[i] = &Type{r.schema.Types[name]}
	}
	return l
}

func (r *Schema) Directives() []*Directive {
	var names []string
	for name := range r.schema.DeclaredDirectives {
		names = append(names, name)
	}
	sort.Strings(names)

	l := make([]*Directive, len(names))
	for i, name := range names {
		l[i] = &Directive{r.schema.DeclaredDirectives[name]}
	}
	return l
}

func (r *Schema) QueryType() *Type {
	t, ok := r.schema.EntryPoints["query"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) MutationType() *Type {
	t, ok := r.schema.EntryPoints["mutation"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) SubscriptionType() *Type {
	t, ok := r.schema.EntryPoints["subscription"]
	if !ok {
		return nil
	}
	return &Type{t}
}

type Type struct {
	typ schema.Type
}

// WrapType is only used internally.
func WrapType(typ schema.Type) *Type {
	return &Type{typ}
}

func (t *Type) To__Type() (*Type, bool) {
	return t, true
}

func (r *Type) Kind() string {
	return r.typ.Kind()
}

func (r *Type) Name() *string {
	if named, ok := r.typ.(schema.NamedType); ok {
		name := named.TypeName()
		return &name
	}
	return nil
}

func (r *Type) Description() *string {
	if named, ok := r.typ.(schema.NamedType); ok {
		desc := named.Description()
		if desc == "" {
			return nil
		}
		return &desc
	}
	return nil
}

func (r *Type) Fields(args *struct{ IncludeDeprecated bool }) *[]*Field {
	var fields schema.FieldList
	switch t := r.typ.(type) {
	case *schema.Object:
		fields = t.Fields
	case *schema.Interface:
		fields = t.Fields
	default:
		return nil
	}

	var l []*Field
	for _, f := range fields {
		if d := f.Directives.Get("deprecated"); d == nil || args.IncludeDeprecated {
			l = append(l, &Field{f})
		}
	}
	return &l
}

func (r *Type) Interfaces() *[]*Type {
	t, ok := r.typ.(*schema.Object)
	if !ok {
		return nil
	}

	l := make([]*Type, len(t.Interfaces))
	for i, intf := range t.Interfaces {
		l[i] = &Type{intf}
	}
	return &l
}

func (r *Type) PossibleTypes() *[]*Type {
	var possibleTypes []*schema.Object
	switch t := r.typ.(type) {
	case *schema.Interface:
		possibleTypes = t.PossibleTypes
	case *schema.Union:
		possibleTypes = t.PossibleTypes
	default:
		return nil
	}

	sort.Slice(possibleTypes, func(i, j int) bool {
		return possibleTypes[i].Name < possibleTypes[j].Name
	})

	l := make([]*Type, len(possibleTypes))
	for i, intf := range possibleTypes {
		l[i] = &Type{intf}
	}
	return &l
}

func (r *Type) EnumValues(args *struct{ IncludeDeprecated bool }) *[]*EnumValue {
	t, ok := r.typ.(*schema.Enum)
	if !ok {
		return nil
	}

	var l []*EnumValue
	for _, v := range t.Values {
		if d := v.Directives.Get("deprecated"); d == nil || args.IncludeDeprecated {
			l = append(l, &EnumValue{v})
		}
	}
	return &l
}

func (r *Type) InputFields() *[]*InputValue {
	t, ok := r.typ.(*schema.InputObject)
	if !ok {
		return nil
	}

	l := make([]*InputValue, len(t.Fields))
	for i, v := range t.Fields {
		l[i] = &InputValue{v}
	}
	return &l
}

func (r *Type) OfType() *Type {
	switch t := r.typ.(type) {
	case *schema.List:
		return &Type{t.OfType}
	case *schema.NonNull:
		return &Type{t.OfType}
	default:
		return nil
	}
}

type Field struct {
	field *schema.Field
}

func (r *Field) Name() string {
	return r.field.Name
}

func (r *Field) Description() *string {
	return Description(r.field.Desc)
}

func Description(desc schema.Description) *string {
	if desc.ShowType == lexer.NoDescription {
		return nil
	}
	if desc.ShowType == lexer.PossibleDescription {
		if desc.Text == "" {
			return nil
		}
	}
	return &desc.Text
}

func (r *Field) Args() []*InputValue {
	l := make([]*InputValue, len(r.field.Args))
	for i, v := range r.field.Args {
		l[i] = &InputValue{v}
	}
	return l
}

func (r *Field) Type() *Type {
	return &Type{r.field.Type}
}

func (r *Field) IsDeprecated() bool {
	return r.field.Directives.Get("deprecated") != nil
}

func (r *Field) DeprecationReason() *string {
	d := r.field.Directives.Get("deprecated")
	if d == nil {
		return nil
	}
	reason := d.Args.MustGet("reason").Evaluate(nil).(string)
	return &reason
}

type InputValue struct {
	value *schema.InputValue
}

func (r *InputValue) To__InputValue() (*InputValue, bool) {
	return r, true
}

func (r *InputValue) Name() string {
	return r.value.Name
}

func (r *InputValue) Description() *string {
	return Description(r.value.Desc)
}

func (r *InputValue) Type() *Type {
	return &Type{r.value.Type}
}

func (r *InputValue) DefaultValue() *string {
	if r.value.Default == nil {
		return nil
	}
	s := r.value.Default.String()
	return &s
}

type EnumValue struct {
	value *schema.EnumValue
}

func (r *EnumValue) Name() string {
	return r.value.Name
}

func (r *EnumValue) Description() *string {
	return Description(r.value.Desc)
}

func (r *EnumValue) IsDeprecated() bool {
	return r.value.Directives.Get("deprecated") != nil
}

func (r *EnumValue) DeprecationReason() *string {
	d := r.value.Directives.Get("deprecated")
	if d == nil {
		return nil
	}
	reason := d.Args.MustGet("reason").Evaluate(nil).(string)
	return &reason
}

type Directive struct {
	directive *schema.DirectiveDecl
}

func (r *Directive) Name() string {
	return r.directive.Name
}

func (r *Directive) Description() *string {
	return Description(r.directive.Desc)
}

func (r *Directive) Locations() []string {
	return r.directive.Locs
}

func (r *Directive) Args() []*InputValue {
	l := make([]*InputValue, len(r.directive.Args))
	for i, v := range r.directive.Args {
		l[i] = &InputValue{v}
	}
	return l
}

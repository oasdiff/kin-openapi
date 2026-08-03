package openapi3

// Generated. Each method is the same three steps: decode into a shadow type so
// the decoder does not recurse into this method, collect the keys the struct
// does not declare as extensions, and read the origin off the node.

import (
	yaml "go.yaml.in/yaml/v3"
)

func (components *Components) UnmarshalYAML(node *yaml.Node) error {
	type ComponentsBis Components
	var x ComponentsBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*components = Components(x)
	setChildOriginKeys(node, components, nativeOriginFile())
	return nil
}

func (contact *Contact) UnmarshalYAML(node *yaml.Node) error {
	type ContactBis Contact
	var x ContactBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*contact = Contact(x)
	setChildOriginKeys(node, contact, nativeOriginFile())
	return nil
}

func (discriminator *Discriminator) UnmarshalYAML(node *yaml.Node) error {
	type DiscriminatorBis Discriminator
	var x DiscriminatorBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*discriminator = Discriminator(x)
	setChildOriginKeys(node, discriminator, nativeOriginFile())
	return nil
}

func (encoding *Encoding) UnmarshalYAML(node *yaml.Node) error {
	type EncodingBis Encoding
	var x EncodingBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*encoding = Encoding(x)
	setChildOriginKeys(node, encoding, nativeOriginFile())
	return nil
}

func (example *Example) UnmarshalYAML(node *yaml.Node) error {
	type ExampleBis Example
	var x ExampleBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*example = Example(x)
	setChildOriginKeys(node, example, nativeOriginFile())
	return nil
}

func (e *ExternalDocs) UnmarshalYAML(node *yaml.Node) error {
	type ExternalDocsBis ExternalDocs
	var x ExternalDocsBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*e = ExternalDocs(x)
	setChildOriginKeys(node, e, nativeOriginFile())
	return nil
}

func (info *Info) UnmarshalYAML(node *yaml.Node) error {
	type InfoBis Info
	var x InfoBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*info = Info(x)
	setChildOriginKeys(node, info, nativeOriginFile())
	return nil
}

func (license *License) UnmarshalYAML(node *yaml.Node) error {
	type LicenseBis License
	var x LicenseBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*license = License(x)
	setChildOriginKeys(node, license, nativeOriginFile())
	return nil
}

func (link *Link) UnmarshalYAML(node *yaml.Node) error {
	type LinkBis Link
	var x LinkBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*link = Link(x)
	setChildOriginKeys(node, link, nativeOriginFile())
	return nil
}

func (mediaType *MediaType) UnmarshalYAML(node *yaml.Node) error {
	type MediaTypeBis MediaType
	var x MediaTypeBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*mediaType = MediaType(x)
	setChildOriginKeys(node, mediaType, nativeOriginFile())
	return nil
}

func (doc *T) UnmarshalYAML(node *yaml.Node) error {
	type TBis T
	var x TBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*doc = T(x)
	setChildOriginKeys(node, doc, nativeOriginFile())
	return nil
}

func (operation *Operation) UnmarshalYAML(node *yaml.Node) error {
	type OperationBis Operation
	var x OperationBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*operation = Operation(x)
	setChildOriginKeys(node, operation, nativeOriginFile())
	return nil
}

func (parameter *Parameter) UnmarshalYAML(node *yaml.Node) error {
	type ParameterBis Parameter
	var x ParameterBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*parameter = Parameter(x)
	setChildOriginKeys(node, parameter, nativeOriginFile())
	return nil
}

func (pathItem *PathItem) UnmarshalYAML(node *yaml.Node) error {
	type PathItemBis PathItem
	var x PathItemBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*pathItem = PathItem(x)
	setChildOriginKeys(node, pathItem, nativeOriginFile())
	return nil
}

func (requestBody *RequestBody) UnmarshalYAML(node *yaml.Node) error {
	type RequestBodyBis RequestBody
	var x RequestBodyBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*requestBody = RequestBody(x)
	setChildOriginKeys(node, requestBody, nativeOriginFile())
	return nil
}

func (response *Response) UnmarshalYAML(node *yaml.Node) error {
	type ResponseBis Response
	var x ResponseBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*response = Response(x)
	setChildOriginKeys(node, response, nativeOriginFile())
	return nil
}

func (schema *Schema) UnmarshalYAML(node *yaml.Node) error {
	type SchemaBis Schema
	var x SchemaBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*schema = Schema(x)
	setChildOriginKeys(node, schema, nativeOriginFile())
	return nil
}

func (ss *SecurityScheme) UnmarshalYAML(node *yaml.Node) error {
	type SecuritySchemeBis SecurityScheme
	var x SecuritySchemeBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*ss = SecurityScheme(x)
	setChildOriginKeys(node, ss, nativeOriginFile())
	return nil
}

func (flows *OAuthFlows) UnmarshalYAML(node *yaml.Node) error {
	type OAuthFlowsBis OAuthFlows
	var x OAuthFlowsBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*flows = OAuthFlows(x)
	setChildOriginKeys(node, flows, nativeOriginFile())
	return nil
}

func (flow *OAuthFlow) UnmarshalYAML(node *yaml.Node) error {
	type OAuthFlowBis OAuthFlow
	var x OAuthFlowBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*flow = OAuthFlow(x)
	setChildOriginKeys(node, flow, nativeOriginFile())
	return nil
}

func (server *Server) UnmarshalYAML(node *yaml.Node) error {
	type ServerBis Server
	var x ServerBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*server = Server(x)
	setChildOriginKeys(node, server, nativeOriginFile())
	return nil
}

func (serverVariable *ServerVariable) UnmarshalYAML(node *yaml.Node) error {
	type ServerVariableBis ServerVariable
	var x ServerVariableBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*serverVariable = ServerVariable(x)
	setChildOriginKeys(node, serverVariable, nativeOriginFile())
	return nil
}

func (tag *Tag) UnmarshalYAML(node *yaml.Node) error {
	type TagBis Tag
	var x TagBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*tag = Tag(x)
	setChildOriginKeys(node, tag, nativeOriginFile())
	return nil
}

func (xml *XML) UnmarshalYAML(node *yaml.Node) error {
	type XMLBis XML
	var x XMLBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile())
	*xml = XML(x)
	setChildOriginKeys(node, xml, nativeOriginFile())
	return nil
}

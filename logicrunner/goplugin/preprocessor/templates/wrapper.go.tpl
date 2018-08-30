package {{ .PackageName}}

import (
    "{{.FoundationPath}}"
)

{{ range $method := .Methods }}
func (self *{{ $.ContractType }}) INSWRAPER_{{ $method.Name }}(cbor foundation.CBORMarshaler, data []byte) ([]byte) {
    {{ $method.ArgumentsZeroList }}
    cbor.Unmarshal(&args, data)

    {{ $method.Results }}:= self.{{ $method.Name }}( {{ $method.Arguments }} )

    return cbor.Marshal([]interface{} { {{ $method.Results }}} )
}
{{ end }}

var INSEXPORT {{ .ContractType }}
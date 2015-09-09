package skl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/swiftkick-io/namedtuple"
	"github.com/swiftkick-io/namedtuple/schema"
)

func TestEncoding(t *testing.T) {

	data, _ := ioutil.ReadFile("/Users/mfranks/.go/src/github.com/subsilent/kappa/skl/skl.nt")
	// fmt.Println(string(data))
	pkgList := schema.NewPackageList()

	// create parser
	parser := schema.NewParser(pkgList)
	pkg, _ := parser.Parse("skl", string(data))
	// fmt.Println(err)
	// fmt.Println(pkg)

	// Builders
	buf := make([]byte, 16384)
	builders := make(map[string]namedtuple.TupleBuilder)

	for _, typ := range pkg.Types {
		tupleType := namedtuple.New(pkg.Name, typ.Name)
		for _, v := range typ.Versions {
			fields := make([]namedtuple.Field, 0)
			for _, f := range v.Fields {
				field := namedtuple.Field{}
				field.Name = f.Name
				field.Required = f.IsRequired

				switch f.Type {
				case "string":
					if f.IsArray {
						field.Type = namedtuple.StringField
					} else {
						field.Type = namedtuple.StringArrayField
					}
				case "byte":
					if f.IsArray {
						field.Type = namedtuple.Uint8Field
					} else {
						field.Type = namedtuple.Uint8ArrayField
					}
				case "uint8":
					if f.IsArray {
						field.Type = namedtuple.Uint8Field
					} else {
						field.Type = namedtuple.Uint8ArrayField
					}
				case "int8":
					if f.IsArray {
						field.Type = namedtuple.Int8Field
					} else {
						field.Type = namedtuple.Int8ArrayField
					}
				case "uint16":
					if f.IsArray {
						field.Type = namedtuple.Uint16Field
					} else {
						field.Type = namedtuple.Uint16ArrayField
					}
				case "int16":
					if f.IsArray {
						field.Type = namedtuple.Int16Field
					} else {
						field.Type = namedtuple.Int16ArrayField
					}
				case "uint32":
					if f.IsArray {
						field.Type = namedtuple.Uint32Field
					} else {
						field.Type = namedtuple.Uint32ArrayField
					}
				case "int32":
					if f.IsArray {
						field.Type = namedtuple.Int32Field
					} else {
						field.Type = namedtuple.Int32ArrayField
					}
				case "uint64":
					if f.IsArray {
						field.Type = namedtuple.Uint64Field
					} else {
						field.Type = namedtuple.Uint64ArrayField
					}
				case "int64":
					if f.IsArray {
						field.Type = namedtuple.Int64Field
					} else {
						field.Type = namedtuple.Int64ArrayField
					}
				case "float32":
					if f.IsArray {
						field.Type = namedtuple.Float32Field
					} else {
						field.Type = namedtuple.Float32ArrayField
					}
				case "float64":
					if f.IsArray {
						field.Type = namedtuple.Float64Field
					} else {
						field.Type = namedtuple.Float64ArrayField
					}
				case "timestamp":
					if f.IsArray {
						field.Type = namedtuple.TimestampField
					} else {
						field.Type = namedtuple.TimestampArrayField
					}
				case "tuple":
					if f.IsArray {
						field.Type = namedtuple.TupleField
					} else {
						field.Type = namedtuple.TupleArrayField
					}
				case "int":
					if f.IsArray {
						field.Type = namedtuple.Int64Field
					} else {
						field.Type = namedtuple.Int64ArrayField
					}
				case "float":
					if f.IsArray {
						field.Type = namedtuple.Float64Field
					} else {
						field.Type = namedtuple.Float64ArrayField
					}
				case "bool":
					if f.IsArray {
						field.Type = namedtuple.BooleanField
					} else {
						field.Type = namedtuple.BooleanArrayField
					}
				default:
					if f.IsArray {
						field.Type = namedtuple.TupleField
					} else {
						field.Type = namedtuple.TupleArrayField
					}
				}

				fields = append(fields, field)
			}

			tupleType.AddVersion(fields...)
		}

		builders[pkg.Name+"."+typ.Name] = namedtuple.NewBuilder(tupleType, buf)
		// typ
	}

	// Get builder
	b := builders["skl.StatusCode"]
	b.PutUint16("code", 1)
	b.PutString("message", "This is a message")
	tup, err := b.Build()
	fmt.Println(tup.Size())
	fmt.Println(tup)
	fmt.Println(err)

	var buffer bytes.Buffer
	n, err := tup.WriteTo(&buffer)
	fmt.Println(n)
	fmt.Println(err)
	fmt.Println(buffer.Bytes())
	// fmt.Println(buf)
}

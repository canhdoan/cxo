package skyobjects

import (
	"bytes"
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Container contains skyobjects.
type Container struct {
	ds      *data.DB
	rootKey cipher.SHA256
	rootSeq uint64
	schemas map[cipher.SHA256]string
}

// NewContainer creates a new skyobjects container.
func NewContainer(ds *data.DB) (c *Container) {
	c = &Container{
		ds:      ds,
		schemas: make(map[cipher.SHA256]string),
	}
	// TODO: Register default schemas in container.
	return
}

// Save saves an object into container.
func (c *Container) Save(schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	// TODO: Special cases for HashRoot and HashArray.
	h := href{SchemaKey: schemaKey, Data: data}
	key = c.ds.AddAutoKey(encoder.Serialize(h))
	return
}

// SaveSchema saves a schema to container.
func (c *Container) SaveSchema(object interface{}) (schemaKey cipher.SHA256) {
	schema := ReadSchema(object)
	schemaData := encoder.Serialize(schema)
	h := href{SchemaKey: _schemaType, Data: schemaData}
	schemaKey = c.ds.AddAutoKey(encoder.Serialize(h))

	// Append data to c.schemas
	c.schemas[schemaKey] = schema.Name
	return
}

// Get retrieves a stored object.
func (c *Container) Get(key cipher.SHA256) (schemaKey cipher.SHA256, data []byte, e error) {
	hrefData, ok := c.ds.Get(key)
	if ok == false {
		e = fmt.Errorf("no object found with key '%s'", key.Hex())
		return
	}
	var h href
	encoder.DeserializeRaw(hrefData, &h) // Shouldn't create an error, everything stored in db is of type href.
	schemaKey, data = h.SchemaKey, h.Data
	return
}

// GetAllOfSchema gets all keys of objects with specified schemaKey.
func (c *Container) GetAllOfSchema(schemaKey cipher.SHA256) (objKeys []cipher.SHA256) {
	query := func(key cipher.SHA256, data []byte) bool {
		return bytes.Compare(schemaKey[:32], data[:32]) == 0
	}
	return c.ds.Where(query)
}

# mgoutil
--
    import "github.com/juju/mgoutil"

Package mgoutil provides helper functions relating to the mgo package.

## Usage

#### type Update

```go
type Update struct {
	// Set holds the fields to be set keyed by field name.
	Set map[string]interface{} `bson:"$set,omitempty"`

	// Unset holds the fields to be unset keyed by field name. Note that
	// the key values will be ignored.
	Unset map[string]interface{} `bson:"$unset,omitempty"`
}
```

Update represents a document update operation. When marshaled and provided to an
update operation, it will set all the fields in Set and unset all the fields in
Unset.

#### func  AsUpdate

```go
func AsUpdate(x interface{}) (Update, error)
```
AsUpdate returns the given object as an Update value holding all the fields of
x, which must be acceptable to bson.Marshal, with zero-valued omitempty fields
returned in Unset and others returned in Set. On success, the returned Set and
Unset fields will always be non-nil, even when they contain no items.

Note that the _id field is omitted, as it is not possible to set this in an
update operation.

This can be useful where an update operation is required to update only some
subset of a given document without hard-coding all the struct fields into the
update document.

For example,

    u, err := AsUpdate(x)
    if err != nil {
    	...
    }
    coll.UpdateId(id, u)

is equivalent to:

    coll.UpdateId(id, x)

as long as all the fields in the database document are mentioned in x. If there
are other fields stored, they won't be affected.

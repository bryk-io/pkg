package orm

import (
	"bytes"
	"text/template"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var tplUpdateDocument *template.Template
var updateDocument = `[{
	"$match": {
		"operationType": "update",
		"fullDocument._id": {{.ID}}
	}
}]`

// Parse built-in templates.
func init() {
	tplUpdateDocument, _ = template.New("updateID").Parse(updateDocument)
}

// PipelineCollection is a helper function to setup a pipeline to
// receive all change events for a specific MongoDB collection.
func PipelineCollection() mongo.Pipeline {
	return mongo.Pipeline{}
}

// PipelineUpdateDocument returns a pipeline to receive update notifications
// for a specific document based on its '_id' field.
func PipelineUpdateDocument(oid primitive.ObjectID) mongo.Pipeline {
	var doc mongo.Pipeline
	buf := bytes.NewBuffer(nil)
	_ = tplUpdateDocument.Execute(buf, map[string]interface{}{"ID": oid})
	_ = bson.UnmarshalExtJSON(buf.Bytes(), false, &doc)
	return doc
}

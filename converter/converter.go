package converter

type Converter interface {
	Convert(message string) error
}

type SQLConverter struct {
	sqlQuery string
}

func NewSQLConverter() Converter {
	return &SQLConverter{}
}

func (converter *SQLConverter) Convert(message string) error {
	// GET DB SCHEMA FOR RAG CONTEXT

	// QUERY OPENAI API TO GENERATE QUERY USING DBSCHEMA

	converter.sqlQuery = message
	return nil
}
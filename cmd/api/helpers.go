package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	_ "github.com/lib/pq"
)

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope) error {
	js, err := json.Marshal(data)

	if err != nil {
		return err
	}

	js = append(js, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil 
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)

	if err != nil {
		// 	json.SyntaxError
		// io.ErrUnexpectedEOF // There is a syntax problem with the JSON being decoded.
		// json.UnmarshalTypeError A JSON value is not appropriate for the destination Go type.
		// json.InvalidUnmarshalError The decode destination is not valid (usually because it is not a
		// pointer). This is actually a problem with our application code,
		// not the JSON itself.
		// io.EOF The JSON being decoded is empty.
		var jsonSyntaxErr *json.SyntaxError
		var jsonUnmarshalTypeErr *json.UnmarshalTypeError
		var jsonInvalideUnmarshalErr *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &jsonSyntaxErr):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", jsonSyntaxErr.Offset)
		case errors.As(err, &jsonUnmarshalTypeErr):
			if jsonUnmarshalTypeErr.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", jsonUnmarshalTypeErr.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %q)", jsonUnmarshalTypeErr.Offset)
		case errors.As(err, &jsonInvalideUnmarshalErr):
			panic(err)
		case errors.Is(err, io.EOF):
			return fmt.Errorf("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		default:
			return err
		}
	}

	// Decode is aim to read streams, so the client can pass multipe json and parse this json { title: "Title" } { title: "Title2"}
	err = dec.Decode(&struct{}{})

	if err != io.EOF {
		return errors.New("body must only contain a single value")
	}

	return nil
}


package pathio

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
)

// MockClient mocks out an S3 bucket
type MockClient struct {
	// Filesystem holds a theoretical S3 bucket for mocking puproses
	Filesystem map[string]string
	// These are errors that will be returned by the mocked methods if set
	WriteErr, ReaderErr, WriteReaderErr error
	// safety because it is easy
	lock sync.Mutex
}

// Reader returns MockClient.ReaderErr if set, otherwise reads from internal data.
func (m *MockClient) Reader(path string) (rc io.ReadCloser, err error) {
	if m.ReaderErr != nil {
		return nil, m.ReaderErr
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	data, exists := m.Filesystem[path]
	if !exists {
		// NOTE: if we decide to read into what the S3 errors are, this will need to be
		// rewritten to return an AWS error.
		return nil, fmt.Errorf("File at '%s' not found", path)
	}
	return ioutil.NopCloser(bytes.NewBufferString(data)), nil
}

// Write returns MockClient.WriteErr if set, otherwise stores the data internally.
func (m *MockClient) Write(path string, input []byte) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Filesystem[path] = string(input)
	return nil
}

// WriteReader returns MockClient.WriteReaderErr if set, otherwise stores the data internally.
func (m *MockClient) WriteReader(path string, input io.ReadSeeker) error {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Filesystem[path] = string(data)
	return nil
}

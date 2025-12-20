package tcp

import (
	"encoding/json"
	"net"
)

type ProgressEmitter struct {
	conn net.Conn
}

func NewProgressEmitter(addr string) (*ProgressEmitter, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &ProgressEmitter{conn: conn}, nil
}

func (e *ProgressEmitter) Emit(update ProgressUpdate) error {
	data, _ := json.Marshal(update)
	_, err := e.conn.Write(append(data, '\n'))
	return err
}

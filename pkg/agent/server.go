// Copyright netdev-cni authors. Apache 2.0 License.
package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// Server listens on a Unix socket and handles CNI binary requests.
type Server struct {
	sockPath   string
	pool       *Pool
	deviceType string
	listener   net.Listener
	mu         sync.Mutex
	done       chan struct{}
}

func NewServer(sockPath string, pool *Pool, deviceType string) *Server {
	return &Server{
		sockPath:   sockPath,
		pool:       pool,
		deviceType: deviceType,
		done:       make(chan struct{}),
	}
}

func (s *Server) ListenAndServe() error {
	os.Remove(s.sockPath)
	l, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.sockPath, err)
	}
	s.mu.Lock()
	s.listener = l
	s.mu.Unlock()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				return err
			}
		}
		go s.handle(conn)
	}
}

func (s *Server) Stop() {
	close(s.done)
	s.mu.Lock()
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Unlock()
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: err.Error()})
		return
	}
	var resp Response
	switch req.Command {
	case "allocate":
		vf, err := s.pool.Allocate(req.PodRef)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.VF = &VFResponse{
				Name:       vf.Name,
				PCIAddress: vf.PCIAddress,
				DeviceType: s.deviceType,
			}
		}
	case "release":
		if err := s.pool.Release(req.PodRef); err != nil {
			resp.Error = err.Error()
		}
	default:
		resp.Error = fmt.Sprintf("unknown command: %s", req.Command)
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

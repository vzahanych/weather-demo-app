package server


type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	return nil
}

func (s *Server) Shutdown() error {
	return nil
}

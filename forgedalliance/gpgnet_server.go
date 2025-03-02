package forgedalliance

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

type GpgNetServer struct {
	port        uint
	tcpSocket   *net.Listener
	currentConn *net.Conn
	state       string
}

func NewGpgNetServer(port uint) *GpgNetServer {
	return &GpgNetServer{
		port:  port,
		state: "disconnected",
	}
}

func (s *GpgNetServer) Listen(gameToAdapter chan *GpgMessage, adapterToGame chan *GpgMessage) error {
	tcpSocket, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(int(s.port)))
	if err != nil {
		return err
	}

	s.tcpSocket = &tcpSocket

	for {
		conn, err := tcpSocket.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		s.currentConn = &conn

		fmt.Println("New client connected:", conn.RemoteAddr())

		// Wrap the connection in a buffered reader.
		bufferReader := bufio.NewReader(conn)
		faStreamreader := NewFaStreamReader(bufferReader)

		go func() {
			for {
				// Read one message from the connection.
				command, err := faStreamreader.ReadString()
				if err != nil {
					fmt.Println("Error parsing command:", conn.RemoteAddr())
					continue
				}

				chunks, err := faStreamreader.ReadChunks()
				if err != nil {
					fmt.Println("Error parsing arguments:", conn.RemoteAddr())
					continue
				}

				unparsedMsg := GenericGpgMessage{
					Command: command,
					Args:    chunks,
				}

				parsedMsg := unparsedMsg.TryParse()

				gameToAdapter <- &parsedMsg
			}
		}()

		go func() {
			bufferedWriter := bufio.NewWriter(conn)
			faStreamWriter := NewFaStreamWriter(bufferedWriter)
			for msg := range gameToAdapter {
				faStreamWriter.WriteMessage(*msg)
			}
		}()
	}
}

func (s *GpgNetServer) Close() {
	(*s.tcpSocket).Close()
}

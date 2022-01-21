package znet

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func ClientTest() {
	fmt.Println("Client Test ... start")
	//3秒之后发起测试请求，给服务端开启服务的机会
	time.Sleep(3 * time.Second)

	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Println("client start err, exit!")
		return
	}

	for {
		_, err := conn.Write([]byte("Hello Zinx"))
		if err != nil {
			fmt.Println("write error err ", err)
			return
		}
		buf := make([]byte, 10)
		cnt, err := conn.Read(buf)
		if err != nil {
			fmt.Println("read buf error ")
			return
		}

		fmt.Printf(" server call back : %s, cnt = %d\n", buf, cnt)

		time.Sleep(1*time.Second)
	}
}

func TestServer_Serve(t *testing.T) {

	//1
	s := NewServer("[zinx V0.1]")

	go ClientTest()

	//2
	s.Serve()
}

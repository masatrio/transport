package vnet

import (
	"fmt"
	"net"
	"testing"

	"github.com/pion/logging"
	"github.com/stretchr/testify/assert"
)

func TestInterfaces(t *testing.T) {
	log := logging.NewDefaultLoggerFactory().NewLogger("test")

	t.Run("Native", func(t *testing.T) {
		nw := NewNet(nil)
		interfaces, err := nw.Interfaces()
		assert.Nil(t, err, "should succeed")
		log.Debugf("interfaces: %+v\n", interfaces)
		for _, ifc := range interfaces {
			if ifc.Name == "lo0" {
				_, err := ifc.Addrs()
				assert.Nil(t, err, "should succeed")
			}

			if addrs, err := ifc.Addrs(); err == nil {
				for _, addr := range addrs {
					log.Debugf("[%d] %s:%s",
						ifc.Index,
						addr.Network(),
						addr.String())
				}
			}
		}
	})

	t.Run("Virtual", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		for _, ifc := range interfaces {
			switch ifc.Name {
			case "lo0":
				assert.Equal(t, 1, ifc.Index, "Index mismatch")
				assert.Equal(t, 16384, ifc.MTU, "MTU mismatch")
				assert.Equal(t,
					net.HardwareAddr(nil),
					ifc.HardwareAddr,
					"HardwareAddr mismatch")
				assert.Equal(t,
					net.FlagUp|net.FlagLoopback|net.FlagMulticast,
					ifc.Flags,
					"Flags mismatch")

				addrs, err := ifc.Addrs()
				assert.Nil(t, err, "should succeed")
				assert.Equal(t, 1, len(addrs), "should be one address")
			case "eth0":
				assert.Equal(t, 2, ifc.Index, "Index mismatch")
				assert.Equal(t, 1500, ifc.MTU, "MTU mismatch")
				assert.Equal(t, 6, len(ifc.HardwareAddr), "HardwareAddr length mismatch")
				assert.Equal(t,
					net.FlagUp|net.FlagMulticast,
					ifc.Flags,
					"Flags mismatch")

				_, err := ifc.Addrs()
				assert.NotNil(t, err, "should fail")
			default:
				assert.Fail(t, "unknown interface: %v", ifc.Name)
			}

			if addrs, err := ifc.Addrs(); err == nil {
				for _, addr := range addrs {
					log.Debugf("[%d] %s:%s",
						ifc.Index,
						addr.Network(),
						addr.String())
				}
			}
		}
	})

	t.Run("Virtual using InterfaceByName", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		var ifc *Interface

		ifc, err = nw.InterfaceByName("lo0")
		assert.Nil(t, err, "should succeed")
		if ifc.Name == "lo0" {
			assert.Equal(t, 1, ifc.Index, "Index mismatch")
			assert.Equal(t, 16384, ifc.MTU, "MTU mismatch")
			assert.Equal(t,
				net.HardwareAddr(nil),
				ifc.HardwareAddr,
				"HardwareAddr mismatch")
			assert.Equal(t,
				net.FlagUp|net.FlagLoopback|net.FlagMulticast,
				ifc.Flags,
				"Flags mismatch")

			addrs, err2 := ifc.Addrs()
			assert.Nil(t, err2, "should succeed")
			assert.Equal(t, 1, len(addrs), "should be one address")
		}

		ifc, err = nw.InterfaceByName("eth0")
		assert.Nil(t, err, "should succeed")
		assert.Equal(t, 2, ifc.Index, "Index mismatch")
		assert.Equal(t, 1500, ifc.MTU, "MTU mismatch")
		assert.Equal(t, 6, len(ifc.HardwareAddr), "HardwareAddr length mismatch")
		assert.Equal(t,
			net.FlagUp|net.FlagMulticast,
			ifc.Flags,
			"Flags mismatch")

		_, err = ifc.Addrs()
		assert.NotNil(t, err, "should fail")

		_, err = nw.InterfaceByName("foo0")
		assert.NotNil(t, err, "should fail")
	})

	t.Run("Virtual using hasIPAddr", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		var ifc *Interface

		ifc, err = nw.InterfaceByName("eth0")
		assert.Nil(t, err, "should succeed")
		ifc.AddAddr(&net.IPNet{
			IP:   net.ParseIP("10.1.2.3"),
			Mask: net.CIDRMask(24, 32),
		})

		_, err = ifc.Addrs()
		assert.Nil(t, err, "should succeed")

		assert.True(t, nw.v.hasIPAddr(net.ParseIP("127.0.0.1")),
			"the IP addr should exist")
		assert.True(t, nw.v.hasIPAddr(net.ParseIP("10.1.2.3")),
			"the IP addr should exist")
		assert.False(t, nw.v.hasIPAddr(net.ParseIP("192.168.1.1")),
			"the IP addr should NOT exist")
	})

	t.Run("getAllIPAddrs()", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		var ifc *Interface

		ifc, err = nw.InterfaceByName("eth0")
		assert.Nil(t, err, "should succeed")
		ifc.AddAddr(&net.IPNet{
			IP:   net.ParseIP("10.1.2.3"),
			Mask: net.CIDRMask(24, 32),
		})

		ips := nw.v.getAllIPAddrs(false)
		assert.Equal(t, 2, len(ips), "should match")

		for _, ip := range ips {
			log.Debugf("ip: %s", ip.String())
		}
	})

	t.Run("assignPort()", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		addr := "1.2.3.4"
		start := 1000
		end := 1002
		space := end + 1 - start

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		var ifc *Interface

		ifc, err = nw.InterfaceByName("eth0")
		assert.Nil(t, err, "should succeed")
		ifc.AddAddr(&net.IPNet{
			IP:   net.ParseIP(addr),
			Mask: net.CIDRMask(24, 32),
		})

		// attempt to assign port with start > end should fail
		_, err = nw.v.assignPort(net.ParseIP(addr), 3000, 2999)
		assert.NotNil(t, err, "should fail")

		for i := 0; i < space; i++ {
			port, err2 := nw.v.assignPort(net.ParseIP(addr), start, end)
			assert.Nil(t, err2, "should succeed")
			log.Debugf("[%d] got port: %d", i, port)

			udpAddr := net.UDPAddr{
				IP:   net.ParseIP(addr),
				Port: port,
			}
			nw.v.udpConns[udpAddr.String()] = nil
		}

		assert.Equal(t, space, len(nw.v.udpConns), "should match")

		// attempt to assign again should fail
		_, err = nw.v.assignPort(net.ParseIP(addr), start, end)
		assert.NotNil(t, err, "should fail")
	})

	t.Run("determineSourceIP()", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		interfaces, err := nw.Interfaces()
		assert.Equal(t, 2, len(interfaces), "should be one interface")
		assert.Nil(t, err, "should succeed")

		var ifc *Interface

		ifc, err = nw.InterfaceByName("eth0")
		assert.Nil(t, err, "should succeed")
		ifc.AddAddr(&net.IPNet{
			IP:   net.ParseIP("1.2.3.4"),
			Mask: net.CIDRMask(24, 32),
		})

		// Any IP turned into non-loopback IP
		anyIP := net.ParseIP("0.0.0.0")
		dstIP := net.ParseIP("27.1.7.135")
		srcIP := nw.v.determineSourceIP(anyIP, dstIP)
		log.Debugf("anyIP: %s => %s", anyIP.String(), srcIP.String())
		assert.NotNil(t, srcIP, "shouldn't be nil")
		assert.Equal(t, srcIP.String(), "1.2.3.4", "use non-loopback IP")

		// Any IP turned into loopback IP
		anyIP = net.ParseIP("0.0.0.0")
		dstIP = net.ParseIP("127.0.0.2")
		srcIP = nw.v.determineSourceIP(anyIP, dstIP)
		log.Debugf("anyIP: %s => %s", anyIP.String(), srcIP.String())
		assert.NotNil(t, srcIP, "shouldn't be nil")
		assert.Equal(t, srcIP.String(), "127.0.0.1", "use loopback IP")

		// Non any IP won't change
		anyIP = net.ParseIP("1.2.3.4")
		dstIP = net.ParseIP("127.0.0.2")
		srcIP = nw.v.determineSourceIP(anyIP, dstIP)
		log.Debugf("anyIP: %s => %s", anyIP.String(), srcIP.String())
		assert.NotNil(t, srcIP, "shouldn't be nil")
		assert.True(t, srcIP.Equal(anyIP), "IP change")
	})
}

func TestNativeUDP(t *testing.T) {
	log := logging.NewDefaultLoggerFactory().NewLogger("test")

	t.Run("ListenPacket", func(t *testing.T) {
		nw := NewNet(nil)

		conn, err := nw.ListenPacket("udp", "127.0.0.1:0")
		if !assert.Nil(t, err, "should succeed") {
			return
		}

		udpConn, ok := conn.(*net.UDPConn)
		assert.True(t, ok, "should succeed")
		log.Debugf("udpConn: %+v", udpConn)

		laddr := conn.LocalAddr().String()
		log.Debugf("laddr: %s", laddr)
	})
}

func TestVirtualUDPListen(t *testing.T) {
	loggerFactory := logging.NewDefaultLoggerFactory()
	log := loggerFactory.NewLogger("test")

	t.Run("ListenPacket random port", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		conn, err := nw.ListenPacket("udp", "127.0.0.1:0")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr().String()
		log.Debugf("laddr: %s", laddr)

		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})

	t.Run("ListenPacket specific port", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		conn, err := nw.ListenPacket("udp", "127.0.0.1:50916")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr().String()
		assert.Equal(t, "127.0.0.1:50916", laddr, "should match")

		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})

	t.Run("ListenUDP random port", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		srcAddr := &net.UDPAddr{
			IP: net.ParseIP("127.0.0.1"),
		}
		conn, err := nw.ListenUDP("udp", srcAddr)
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr().String()
		log.Debugf("laddr: %s", laddr)

		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})

	t.Run("ListenUDP specific port", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		srcAddr := &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 60916,
		}
		conn, err := nw.ListenUDP("udp", srcAddr)
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr().String()
		assert.Equal(t, "127.0.0.1:60916", laddr, "should match")

		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})
}

func TestVirtualUDPDial(t *testing.T) {
	loggerFactory := logging.NewDefaultLoggerFactory()
	log := loggerFactory.NewLogger("test")

	t.Run("Simple lo0", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		conn, err := nw.Dial("udp", "127.0.0.1:1234")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr()
		log.Debugf("laddr: %s", laddr.String())

		raddr := conn.RemoteAddr()
		log.Debugf("raddr: %s", raddr.String())

		assert.Equal(t, "127.0.0.1", laddr.(*net.UDPAddr).IP.String(), "should match")
		assert.True(t, laddr.(*net.UDPAddr).Port != 0, "should match")
		assert.Equal(t, "127.0.0.1:1234", raddr.String(), "should match")
		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})

	t.Run("Simple eth0", func(t *testing.T) {
		wan, err := NewRouter(&RouterConfig{
			CIDR:          "1.2.3.0/24",
			LoggerFactory: loggerFactory,
		})
		assert.Nil(t, err, "should succeed")
		assert.NotNil(t, wan, "should succeed")

		nw := NewNet(&NetConfig{})

		assert.NoError(t, wan.AddNet(nw), "should succeed")

		conn, err := nw.Dial("udp", "27.3.4.5:1234")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr()
		log.Debugf("laddr: %s", laddr.String())

		raddr := conn.RemoteAddr()
		log.Debugf("raddr: %s", raddr.String())

		assert.Equal(t, "1.2.3.1", laddr.(*net.UDPAddr).IP.String(), "should match")
		assert.True(t, laddr.(*net.UDPAddr).Port != 0, "should match")
		assert.Equal(t, "27.3.4.5:1234", raddr.String(), "should match")
		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})

	t.Run("Using resolver", func(t *testing.T) {
		wan, err := NewRouter(&RouterConfig{
			CIDR:          "1.2.3.0/24",
			LoggerFactory: loggerFactory,
		})
		assert.Nil(t, err, "should succeed")
		assert.NotNil(t, wan, "should succeed")

		wan.AddHost("test.pion.ly", net.ParseIP("30.31.32.33"))

		nw := NewNet(&NetConfig{})

		assert.NoError(t, wan.AddNet(nw), "should succeed")

		conn, err := nw.Dial("udp", "test.pion.ly:1234")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr()
		log.Debugf("laddr: %s", laddr.String())

		raddr := conn.RemoteAddr()
		log.Debugf("raddr: %s", raddr.String())

		assert.Equal(t, "1.2.3.1", laddr.(*net.UDPAddr).IP.String(), "should match")
		assert.True(t, laddr.(*net.UDPAddr).Port != 0, "should match")
		assert.Equal(t, "30.31.32.33:1234", raddr.String(), "should match")
		assert.Equal(t, 1, len(nw.v.udpConns), "should match")
		assert.Nil(t, conn.Close(), "should succeed")
		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
	})
}

func TestVirtualUDP(t *testing.T) {
	loggerFactory := logging.NewDefaultLoggerFactory()
	log := loggerFactory.NewLogger("test")

	t.Run("Loopback", func(t *testing.T) {
		nw := NewNet(&NetConfig{})

		conn, err := nw.ListenPacket("udp", "127.0.0.1:50916")
		assert.Nil(t, err, "should succeed")

		laddr := conn.LocalAddr()
		assert.Equal(t, "127.0.0.1:50916", laddr.String(), "should match")

		c := newChunkUDP(&net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 4000,
		}, &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 50916,
		})
		c.userData = []byte("Hello!")

		var hasReceived bool
		recvdCh := make(chan bool)
		doneCh := make(chan struct{})

		go func() {
			var err error
			var n int
			var addr net.Addr
			buf := make([]byte, 1500)
			for {
				n, addr, err = conn.ReadFrom(buf)
				if err != nil {
					log.Debugf("ReadFrom returned: %v", err)
					break
				}

				assert.Equal(t, 6, len(c.userData), "should match")
				assert.Equal(t, "127.0.0.1:4000", addr.String(), "should match")
				assert.Equal(t, "Hello!", string(buf[:n]), "should match")

				recvdCh <- true
			}

			close(doneCh)
		}()

		nw.v.onInboundChunk(c)

	loop:
		for {
			select {
			case <-recvdCh:
				hasReceived = true
				assert.Nil(t, conn.Close(), "should succeed")
			case <-doneCh:
				break loop
			}
		}

		assert.Equal(t, 0, len(nw.v.udpConns), "should match")
		assert.True(t, hasReceived, "should have received data")
	})

	t.Run("End-to-End", func(t *testing.T) {
		doneCh := make(chan struct{})

		// WAN
		wan, err := NewRouter(&RouterConfig{
			CIDR:          "1.2.3.0/24",
			LoggerFactory: loggerFactory,
		})
		assert.Nil(t, err, "should succeed")
		assert.NotNil(t, wan, "should succeed")

		net1 := NewNet(&NetConfig{})

		err = wan.AddNet(net1)
		assert.Nil(t, err, "should succeed")
		ip1, err := getIPAddr(net1)
		assert.Nil(t, err, "should succeed")

		net2 := NewNet(&NetConfig{})

		err = wan.AddNet(net2)
		assert.Nil(t, err, "should succeed")
		ip2, err := getIPAddr(net2)
		assert.Nil(t, err, "should succeed")

		conn1, err := net1.ListenPacket(
			"udp",
			fmt.Sprintf("%s:%d", ip1, 1234),
		)
		assert.Nil(t, err, "should succeed")

		conn2, err := net2.ListenPacket(
			"udp",
			fmt.Sprintf("%s:%d", ip2, 5678),
		)
		assert.Nil(t, err, "should succeed")

		// start the router
		err = wan.Start()
		assert.Nil(t, err, "should succeed")

		conn1RcvdCh := make(chan bool)

		// conn1
		go func() {
			buf := make([]byte, 1500)
			for {
				log.Debug("conn1: wait for a message..")
				n, _, err2 := conn1.ReadFrom(buf)
				if err2 != nil {
					log.Debugf("ReadFrom returned: %v", err2)
					break
				}

				log.Debugf("conn1 received %s", string(buf[:n]))
				conn1RcvdCh <- true
			}
			close(doneCh)
		}()

		// conn2
		go func() {
			buf := make([]byte, 1500)
			for {
				log.Debug("conn2: wait for a message..")
				n, addr, err2 := conn2.ReadFrom(buf)
				if err2 != nil {
					log.Debugf("ReadFrom returned: %v", err2)
					break
				}

				log.Debugf("conn2 received %s", string(buf[:n]))

				// echo back to conn1
				nSent, err2 := conn2.WriteTo([]byte("Good-bye!"), addr)
				assert.Nil(t, err2, "should succeed")
				assert.Equal(t, 9, nSent, "should match")
			}
		}()

		log.Debug("conn1: sending")
		nSent, err := conn1.WriteTo(
			[]byte("Hello!"),
			conn2.LocalAddr(),
		)
		assert.Nil(t, err, "should succeed")
		assert.Equal(t, 6, nSent, "should match")

	loop:
		for {
			select {
			case <-conn1RcvdCh:
				assert.NoError(t, conn1.Close(), "should succeed")
				assert.NoError(t, conn2.Close(), "should succeed")
			case <-doneCh:
				break loop
			}
		}

		assert.NoError(t, wan.Stop(), "should succeed")
	})
}
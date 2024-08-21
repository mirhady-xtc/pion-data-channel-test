package main

import "C"
import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type SignalingData struct {
	clientSDP1 *bytes.Buffer
	clientSDP2 *bytes.Buffer
	candidate1 *bytes.Buffer
	candidate2 *bytes.Buffer
}

var (
	dataChannel  *webrtc.DataChannel
	dataChannel1 *webrtc.DataChannel
	dataChannel2 *webrtc.DataChannel
	dataChannel3 *webrtc.DataChannel
	dataChannel4 *webrtc.DataChannel
	dataChannel5 *webrtc.DataChannel

	useWSS bool

	dataChannelOpened  = false
	dataChannel1Opened = false

	messageQueue  chan []byte
	messageQueue1 chan []byte
	clientStats   chan []byte

	notifyWebSocket sync.Mutex
	cond            *sync.Cond
)

func SendData(data *C.char, dataLen C.int, channelIndex C.int) {
	goData := C.GoBytes(unsafe.Pointer(data), dataLen)

	if !dataChannelOpened || !dataChannel1Opened {
		return
	}

	switch channelIndex {
	case 0:
		if dataChannel != nil {
			if err := dataChannel.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel: %v\n", err)
			}
		} else {
			log.Println("dataChannel is not initialized")
		}
	case 1:
		if dataChannel1 != nil {
			if err := dataChannel1.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel1: %v\n", err)
			}
		} else {
			log.Println("dataChannel1 is not initialized")
		}
	case 2:
		if dataChannel2 != nil {
			if err := dataChannel2.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel2: %v\n", err)
			}
		} else {
			log.Println("dataChannel2 is not initialized")
		}
	case 3:
		if dataChannel3 != nil {
			if err := dataChannel3.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel3: %v\n", err)
			}
		} else {
			log.Println("dataChannel3 is not initialized")
		}
	case 4:
		if dataChannel4 != nil {
			if err := dataChannel4.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel4: %v\n", err)
			}
		} else {
			log.Println("dataChannel4 is not initialized")
		}
	case 5:
		if dataChannel5 != nil {
			if err := dataChannel5.Send(goData); err != nil {
				log.Printf("Error sending data on dataChannel5: %v\n", err)
			}
		} else {
			log.Println("dataChannel5 is not initialized")
		}
	default:
		log.Println("Invalid channel index")
	}
}

func MyMain(bind_ip_c *C.char, bind_ip1_c *C.char, use_interface_filter C.int, interface_1 *C.char, interface_2 *C.char, use_wss_flag C.int, print_flag C.int) {
	cond = sync.NewCond(&notifyWebSocket)

	bind_ip := C.GoString(bind_ip_c)
	bind_ip1 := C.GoString(bind_ip1_c)
	useWSS = use_wss_flag == 1 // toggle this to false for local connection

	// fmt.Println("bind_ip:", bind_ip)
	// fmt.Println("bind_ip1:", bind_ip1)

	messageQueue = make(chan []byte, 1024)
	messageQueue1 = make(chan []byte, 1024)
	clientStats = make(chan []byte, 1)

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	sigData := make(chan SignalingData)
	sigLocalSDP := make(chan *webrtc.SessionDescription)
	sigLocalSDP1 := make(chan *webrtc.SessionDescription)

	clientSDP1 := bytes.NewBufferString("")
	clientSDP2 := bytes.NewBufferString("")
	candidate1 := bytes.NewBufferString("")
	candidate2 := bytes.NewBufferString("")

	http.HandleFunc("/signaling", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("upgrade:", err)
			return
		}

		fmt.Println("Client connected...")

		for i := 0; i < 2; i++ {
			_, message, err := conn.ReadMessage()
			//fmt.Println("message:", string(message))
			if err == nil && message[2] == ':' {
				if message[0] == 's' {
					if message[1] == '1' {
						clientSDP1 = bytes.NewBuffer(message[3:])
					} else if message[1] == '2' {
						clientSDP2 = bytes.NewBuffer(message[3:])
					}
				} else if message[0] == 'c' {
					if message[1] == '1' {
						candidate1 = bytes.NewBuffer(message[3:])
					} else if message[1] == '2' {
						candidate2 = bytes.NewBuffer(message[3:])
					}
				}
			}
		}

		sigData <- SignalingData{clientSDP1, clientSDP2, candidate1, candidate2}

		localDescription := <-sigLocalSDP
		localDescription1 := <-sigLocalSDP1

		if err := conn.WriteMessage(websocket.TextMessage, []byte("1:"+Encode(localDescription))); err != nil {
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte("2:"+Encode(localDescription1))); err != nil {
			return
		}

		// // Read client stats
		// _, stats, err := conn.ReadMessage()
		// if err != nil {
		// 	fmt.Println("Failed to read client stats:", err)
		// 	return
		// }
		// clientStats <- stats

		go func() {
			// Read client stats
			cond.L.Lock()
			cond.Wait()
			cond.L.Unlock()
			for {
				_, stats, err := conn.ReadMessage()
				if err != nil {
					fmt.Println("Failed to read client stats:", err)
					return
				}
				clientStats <- stats
			}
		}()
	})

	srv := &http.Server{Addr: ":64002"}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	for {
		dataChannelOpened = false
		dataChannel1Opened = false

		settings := webrtc.SettingEngine{}
		settings1 := webrtc.SettingEngine{}

		if use_interface_filter == 1 {
			settings.SetIPFilter(func(ip net.IP) bool {
				return ip.String() == bind_ip
			})
			settings1.SetIPFilter(func(ip net.IP) bool {
				return ip.String() == bind_ip1
			})
		} else {
			settings.SetNAT1To1IPs([]string{bind_ip}, webrtc.ICECandidateTypeHost)
			settings1.SetNAT1To1IPs([]string{bind_ip1}, webrtc.ICECandidateTypeHost)
		}

		api := webrtc.NewAPI(webrtc.WithSettingEngine(settings))
		api1 := webrtc.NewAPI(webrtc.WithSettingEngine(settings1))

		config := webrtc.Configuration{}
		config1 := webrtc.Configuration{}

		peerConnection, err := api.NewPeerConnection(config)
		if err != nil {
			panic(err)
		}

		peerConnection1, err1 := api1.NewPeerConnection(config1)
		if err1 != nil {
			panic(err1)
		}

		done := make(chan bool, 2)

		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			if connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateFailed {
				done <- true
			}
		})

		peerConnection1.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			if connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateFailed {
				done <- true
			}
		})

		ordered := false
		// maxPacketLifeTime := uint16(1)
		maxRetransmits := uint16(0)
		dc_init := webrtc.DataChannelInit{
			Ordered:        &ordered,
			MaxRetransmits: &maxRetransmits,
			// MaxPacketLifeTime: &maxPacketLifeTime,
		}

		dataChannel, err = peerConnection.CreateDataChannel("xdpChannel1", &dc_init)
		if err != nil {
			panic(err)
		}

		dataChannel1, err1 = peerConnection1.CreateDataChannel("xdpChannel2", &dc_init)
		if err1 != nil {
			panic(err1)
		}

		dataChannel2, err1 = peerConnection.CreateDataChannel("xdpChannel3", &dc_init)
		if err1 != nil {
			panic(err1)
		}

		dataChannel3, err1 = peerConnection1.CreateDataChannel("xdpChannel4", &dc_init)
		if err1 != nil {
			panic(err1)
		}

		dataChannel4, err1 = peerConnection.CreateDataChannel("xdpChannel5", &dc_init)
		if err1 != nil {
			panic(err1)
		}

		dataChannel5, err1 = peerConnection1.CreateDataChannel("xdpChannel6", &dc_init)
		if err1 != nil {
			panic(err1)
		}

		dataChannel.OnOpen(func() {
			dataChannelOpened = true
			// fmt.Println("datachannel is open")
		})

		dataChannel1.OnOpen(func() {
			dataChannel1Opened = true
			// fmt.Println("datachannel1 is open")
		})

		dataChannel2.OnOpen(func() {
		})

		dataChannel3.OnOpen(func() {
		})

		dataChannel4.OnOpen(func() {
		})

		dataChannel5.OnOpen(func() {
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	// fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
			// }
			messageQueue <- msg.Data
		})

		dataChannel1.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	// fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel1.Label(), string(msg.Data))
			// }
			messageQueue1 <- msg.Data
		})

		dataChannel2.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel2.Label(), string(msg.Data))
			// }
			messageQueue <- msg.Data
		})

		dataChannel3.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel3.Label(), string(msg.Data))
			// }
			messageQueue1 <- msg.Data
		})

		dataChannel4.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel4.Label(), string(msg.Data))
			// }
			messageQueue <- msg.Data
		})

		dataChannel5.OnMessage(func(msg webrtc.DataChannelMessage) {
			// if print_flag != 0 {
			// 	fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel5.Label(), string(msg.Data))
			// }
			messageQueue1 <- msg.Data
		})

		signalingData := <-sigData
		clientSDP1 := signalingData.clientSDP1
		clientSDP2 := signalingData.clientSDP2

		offer := webrtc.SessionDescription{}
		Decode(clientSDP1.String(), &offer)

		offer1 := webrtc.SessionDescription{}
		Decode(clientSDP2.String(), &offer1)

		// Set the remote SessionDescription
		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			panic(err)
		}

		err1 = peerConnection1.SetRemoteDescription(offer1)
		if err1 != nil {
			panic(err1)
		}

		// Create answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		answer1, err1 := peerConnection1.CreateAnswer(nil)
		if err1 != nil {
			panic(err1)
		}

		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
		gatherComplete1 := webrtc.GatheringCompletePromise(peerConnection1)

		if err = peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidate1.String()}); err != nil {
			panic(err)
		}

		if err1 = peerConnection1.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidate2.String()}); err1 != nil {
			panic(err1)
		}

		// Sets the LocalDescription, and starts our UDP listeners
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}

		err1 = peerConnection1.SetLocalDescription(answer1)
		if err1 != nil {
			panic(err1)
		}

		<-gatherComplete
		<-gatherComplete1

		localDescription := peerConnection.LocalDescription()
		localDescription1 := peerConnection1.LocalDescription()

		sigLocalSDP <- localDescription
		sigLocalSDP1 <- localDescription1

		<-done
		<-done

		dataChannel.Close()
		dataChannel1.Close()
		dataChannel2.Close()
		dataChannel3.Close()
		dataChannel4.Close()
		dataChannel5.Close()
		peerConnection.Close()
		peerConnection1.Close()
	}
}

// MustReadStdin blocks until input is received from stdin
func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	// fmt.Println("")

	return in
}

// Encode encodes the input in base64
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

type intSlice []int

func (i *intSlice) String() string {
	return fmt.Sprint(*i)
}

func (i *intSlice) Set(value string) error {
	values := strings.Split(value, ",")
	for _, v := range values {
		var val int
		_, err := fmt.Sscanf(v, "%d", &val)
		if err != nil {
			return err
		}
		*i = append(*i, val)
	}
	return nil
}

func GenerateByteArray(size int, value int32) ([]byte, error) {
	if size < 4 {
		return nil, fmt.Errorf("size must be at least 4 to store the integer")
	}
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, value)
	if err != nil {
		return nil, err
	}
	byteArray := make([]byte, size)
	copy(byteArray[:4], buf.Bytes())
	return byteArray, nil
}

func main() {
	fmt.Println("Starting WebRTC server...")
	bindIP := flag.String("bind_ip", "54.66.197.78", "IP address to bind")
	bindIP1 := flag.String("bind_ip1", "54.66.197.78", "Secondary IP address to bind")
	useInterfaceFilter := flag.Int("use_interface_filter", 0, "Use interface filter (1: yes, 0: no)")
	interface1 := flag.String("interface_1", "", "First network interface")
	interface2 := flag.String("interface_2", "", "Second network interface")
	useWSSFlag := flag.Int("use_wss", 0, "Use WSS (1: yes, 0: no)")
	printFlag := flag.Int("print", 1, "Print flag (1: yes, 0: no)")

	var numMessages, messageSize, messageInterval, dataChannelCount, pathCount, experimentCount intSlice
	flag.Var(&numMessages, "num_messages", "Number of messages to send (can be multiple values)")
	flag.Var(&messageSize, "message_size", "Size of each message in bytes (can be multiple values)")
	flag.Var(&messageInterval, "message_interval", "Interval between messages in milliseconds (can be multiple values)")
	flag.Var(&dataChannelCount, "data_channel_count", "Number of DataChannels for each path (can be multiple values)")
	flag.Var(&pathCount, "path_count", "Number of paths (can be multiple values)")
	flag.Var(&experimentCount, "experiment_count", "Number of experiments (can be multiple values)")

	flag.Parse()

	fmt.Println("num_messages:", numMessages)
	fmt.Println("message_size:", messageSize)
	fmt.Println("message_interval:", messageInterval)
	fmt.Println("data_channel_count:", dataChannelCount)
	fmt.Println("path_count:", pathCount)

	go MyMain(C.CString(*bindIP), C.CString(*bindIP1), C.int(*useInterfaceFilter), C.CString(*interface1), C.CString(*interface2), C.int(*useWSSFlag), C.int(*printFlag))

	fmt.Println("Type something here and press ENTER when you are sure you have opened the browser and the WebRTC connection is established...")
	MustReadStdin()

	cond.L.Lock()
	cond.Signal()
	cond.L.Unlock()

	for ec := 0; ec < experimentCount[0]; ec++ {
		for _, nm := range numMessages {
			for _, ms := range messageSize {
				for _, mi := range messageInterval {
					for _, dcc := range dataChannelCount {
						for _, pc := range pathCount {

							// ticker := time.NewTicker(time.Duration(mi) * time.Millisecond)
							// defer ticker.Stop()

							start_time := time.Now()

							sentMessages := 0

							path1SendNum := 0
							path2SendNum := 0

							msgSentNumPath1 := 0
							msgSentNumPath2 := 0

							if pc > 1 {
								path1SendNum = nm / 2
								path2SendNum = nm - path1SendNum
							} else {
								path1SendNum = nm
							}

							// for range ticker.C {
							for {
								if sentMessages >= nm {
									break
								}

								// var wg sync.WaitGroup
								// var mu sync.Mutex

								// if pc > 1 {
								// 	wg.Add(1)
								// 	go func() {
								// 		defer wg.Done()

								// 		for i := 0; i < dcc; i++ {
								// 			msgSentNumPath2++
								// 			if msgSentNumPath2 > path2SendNum {
								// 				break
								// 			}
								// 			mu.Lock()
								// 			sentMessages++
								// 			mu.Unlock()
								// 			testMessage, _ := GenerateByteArray(ms, int32(sentMessages))
								// 			SendData((*C.char)(unsafe.Pointer(&testMessage[0])), C.int(len(testMessage)), C.int(i*2+1))
								// 		}
								// 	}()
								// }

								if pc > 1 {
									// defer wg.Done()

									for i := 0; i < dcc; i++ {
										msgSentNumPath2++
										if msgSentNumPath2 > path2SendNum {
											break
										}
										// mu.Lock()
										sentMessages++
										// mu.Unlock()
										testMessage, _ := GenerateByteArray(ms, int32(sentMessages))
										SendData((*C.char)(unsafe.Pointer(&testMessage[0])), C.int(len(testMessage)), C.int(i*2+1))
									}
								}

								// time.Sleep(300 * time.Microsecond)

								if sentMessages >= nm {
									break
								}

								for i := 0; i < dcc; i++ {
									msgSentNumPath1++
									if msgSentNumPath1 > path1SendNum {
										break
									}
									// mu.Lock()
									sentMessages++
									// mu.Unlock()
									testMessage, _ := GenerateByteArray(ms, int32(sentMessages))
									SendData((*C.char)(unsafe.Pointer(&testMessage[0])), C.int(len(testMessage)), C.int(i*2))
								}
								// fmt.Println("datachannel 1 buffer size:", dataChannel.BufferedAmount())

								// print timestamp
								// fmt.Println(time.Now().UnixNano())

								// time.Sleep(5 * time.Millisecond)
								// wg.Wait()
							}

							end_time := time.Now()
							elapsed_time := end_time.Sub(start_time)

							fmt.Printf("\n Sent %d messages of size %d bytes each with an interval of %d milliseconds on %d DataChannels/path and %d path in round %d. Time elapsed: %v\n",
								sentMessages, ms, mi, dcc, pc, ec, elapsed_time)

							// Wait for client stats
							stats := <-clientStats

							// Process stats from client
							var clientResults map[string]interface{}
							if err := json.Unmarshal(stats, &clientResults); err != nil {
								fmt.Println("Failed to unmarshal client stats:", err)
								continue
							}

							fmt.Printf("Detailed stats: \n", clientResults)
						}
					}
				}
			}
		}
	}
}

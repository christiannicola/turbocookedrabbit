package main_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"

	"github.com/houseofcat/turbocookedrabbit/consumer"
	"github.com/houseofcat/turbocookedrabbit/models"
	"github.com/houseofcat/turbocookedrabbit/pools"
	"github.com/houseofcat/turbocookedrabbit/publisher"
	"github.com/houseofcat/turbocookedrabbit/topology"
	"github.com/houseofcat/turbocookedrabbit/utils"
)

var Seasoning *models.RabbitSeasoning
var ConnectionPool *pools.ConnectionPool
var ChannelPool *pools.ChannelPool

func TestMain(m *testing.M) { // Load Configuration On Startup
	var err error
	Seasoning, err = utils.ConvertJSONFileToConfig("testseasoning.json")
	if err != nil {
		return
	}
	ConnectionPool, _ = pools.NewConnectionPool(Seasoning.PoolConfig, true)
	ChannelPool, _ = pools.NewChannelPool(Seasoning.PoolConfig, ConnectionPool, true)

	os.Exit(m.Run())
}

func TestReadConfig(t *testing.T) {
	fileNamePath := "testseasoning.json"

	assert.FileExists(t, fileNamePath)

	config, err := utils.ConvertJSONFileToConfig(fileNamePath)

	assert.Nil(t, err)
	assert.NotEqual(t, "", config.PoolConfig.ConnectionPoolConfig.URI, "RabbitMQ URI should not be blank.")
}

func TestBasicPublish(t *testing.T) {
	//defer leaktest.Check(t)() // Fail on leaked goroutines.
	messageCount := 100000

	// Pre-create test messages
	timeStart := time.Now()
	letters := make([]*models.Letter, messageCount)

	for i := 0; i < messageCount; i++ {
		letters[i] = utils.CreateLetter("", fmt.Sprintf("TestQueue-%d", i%10), nil)
	}

	elapsed := time.Since(timeStart)
	fmt.Printf("Time Elapsed Creating Letters: %s\r\n", elapsed)

	// Test
	timeStart = time.Now()
	amqpConn, err := amqp.Dial(Seasoning.PoolConfig.ConnectionPoolConfig.URI)
	if err != nil {
		return
	}

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		return
	}

	for i := 0; i < messageCount; i++ {
		letter := letters[i]
		amqpChan.Publish(
			letter.Envelope.Exchange,
			letter.Envelope.RoutingKey,
			letter.Envelope.Mandatory,
			letter.Envelope.Immediate,
			amqp.Publishing{
				ContentType: letter.Envelope.ContentType,
				Body:        letter.Body,
			},
		)
	}

	elapsed = time.Since(timeStart)
	fmt.Printf("Publish Time: %s\r\n", elapsed)
	fmt.Printf("Rate: %f msg/s\r\n", float64(messageCount)/elapsed.Seconds())

	// TODO: Poll Queues till the message counts are there. Should be messageCount distributed evenly in 10 queues.
}

func TestTLSConnection(t *testing.T) {
	// https://github.com/streadway/amqp/blob/master/examples_test.go
	// This example assume you have a RabbitMQ node running on localhost
	// with TLS enabled.
	//
	// The easiest way to create the CA, certificates and keys required for these
	// examples is by using tls-gen: https://github.com/michaelklishin/tls-gen
	//
	// A comprehensive RabbitMQ TLS guide can be found at
	// http://www.rabbitmq.com/ssl.html
	//
	// Once you have the required TLS files in place, use the following
	// rabbitmq.config example for the RabbitMQ node that you will run on
	// localhost:
	//
	//   [
	//   {rabbit, [
	//     {tcp_listeners, []},     % listens on 127.0.0.1:5672
	//     {ssl_listeners, [5671]}, % listens on 0.0.0.0:5671
	//     {ssl_options, [{cacertfile,"/path/to/your/testca/cacert.pem"},
	//                    {certfile,"/path/to/your/server/cert.pem"},
	//                    {keyfile,"/path/to/your/server/key.pem"},
	//                    {verify,verify_peer},
	//                    {fail_if_no_peer_cert,true}]}
	//     ]}
	//   ].
	//
	//
	// In the above rabbitmq.config example, we are disabling the plain AMQP port
	// and verifying that clients and fail if no certificate is presented.
	//
	// The self-signing certificate authority's certificate (cacert.pem) must be
	// included in the RootCAs to be trusted, otherwise the server certificate
	// will fail certificate verification.
	//
	// Alternatively to adding it to the tls.Config. you can add the CA's cert to
	// your system's root CAs.  The tls package will use the system roots
	// specific to each support OS.  Under OS X, add (drag/drop) cacert.pem
	// file to the 'Certificates' section of KeyChain.app to add and always
	// trust.  You can also add it via the command line:
	//
	//   security add-certificate testca/cacert.pem
	//   security add-trusted-cert testca/cacert.pem
	//
	// If you depend on the system root CAs, then use nil for the RootCAs field
	// so the system roots will be loaded instead.
	//
	// Server names are validated by the crypto/tls package, so the server
	// certificate must be made for the hostname in the URL.  Find the commonName
	// (CN) and make sure the hostname in the URL matches this common name.  Per
	// the RabbitMQ instructions (or tls-gen) for a self-signed cert, this defaults to the
	// current hostname.
	//
	//   openssl x509 -noout -in /path/to/certificate.pem -subject
	//
	// If your server name in your certificate is different than the host you are
	// connecting to, set the hostname used for verification in
	// ServerName field of the tls.Config struct.
	/* 	cfg := new(tls.Config)

	   	// see at the top
	   	cfg.RootCAs = x509.NewCertPool()

	   	if ca, err := ioutil.ReadFile("testca/cacert.pem"); err == nil {
	   		cfg.RootCAs.AppendCertsFromPEM(ca)
	   	}

	   	// Move the client cert and key to a location specific to your application
	   	// and load them here.

	   	if cert, err := tls.LoadX509KeyPair("client/cert.pem", "client/key.pem"); err == nil {
	   		cfg.Certificates = append(cfg.Certificates, cert)
	   	}

	   	// see a note about Common Name (CN) at the top
	   	conn, err := amqp.DialTLS("amqps://server-name-from-certificate/", cfg)

	   	log.Printf("conn: %v, err: %v", conn, err) */
}

func TestReadTopologyConfig(t *testing.T) {
	fileNamePath := "testtopology.json"

	assert.FileExists(t, fileNamePath)

	config, err := utils.ConvertJSONFileToTopologyConfig(fileNamePath)

	assert.Nil(t, err)
	assert.NotEqual(t, 0, len(config.Exchanges))
	assert.NotEqual(t, 0, len(config.Queues))
	assert.NotEqual(t, 0, len(config.QueueBindings))
	assert.NotEqual(t, 0, len(config.ExchangeBindings))
}

func TestCreateTopologyFromTopologyConfig(t *testing.T) {

	fileNamePath := "testtopology.json"
	assert.FileExists(t, fileNamePath)

	topologyConfig, err := utils.ConvertJSONFileToTopologyConfig(fileNamePath)
	assert.NoError(t, err)

	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, false)
	assert.NoError(t, err)

	topologer := topology.NewTopologer(channelPool)
	assert.NoError(t, err)

	err = topologer.BuildToplogy(topologyConfig, true)
	assert.NoError(t, err)
}

func TestUnbindQueue(t *testing.T) {

	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, false)
	assert.NoError(t, err)

	topologer := topology.NewTopologer(channelPool)
	assert.NoError(t, err)

	err = topologer.UnbindQueue("QueueAttachedToExch01", "RoutingKey1", "MyTestExchange.Child01", nil)
	assert.NoError(t, err)
}

func TestPublishConsumeAckForDuration(t *testing.T) {

	timeDuration := time.Duration(30 * time.Minute)
	timeOut := time.After(timeDuration)
	fmt.Printf("Benchmark Starts: %s\r\n", time.Now())
	fmt.Printf("Est. Benchmark End: %s\r\n", time.Now().Add(timeDuration))

	publisher, _ := publisher.NewPublisher(Seasoning, ChannelPool, nil)
	consumerConfig, _ := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-Ackable"]
	consumer, _ := consumer.NewConsumerFromConfig(consumerConfig, ChannelPool)

	publisher.StartAutoPublish(false)

	letter := utils.CreateLetter("", "ConsumerTestQueue", nil)

	go func() {
	PublishLoop:
		for {
			select {
			case <-timeOut:
				break PublishLoop
			default:
				newLetter := models.Letter(*letter)
				publisher.QueueLetter(&newLetter)
				fmt.Printf("%s: Letter Queued - LetterID: %d\r\n", time.Now(), newLetter.LetterID)
				letter.LetterID++
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	consumer.StartConsuming()

	messagesReceived := 0
	messagesPublished := 0
	messagesFailedToPublish := 0
	messagesAcked := 0
	messagesFailedToAck := 0
	consumerErrors := 0
	channelPoolErrors := 0
	connectionPoolErrors := 0

	// Stop RabbitMQ server after entering loop, then start it again, to test reconnectivity.
ConsumeLoop:
	for {
		select {
		case <-timeOut:
			break ConsumeLoop
		case notice := <-publisher.Notifications():
			if notice.Success {
				fmt.Printf("%s: Published Success - LetterID: %d\r\n", time.Now(), notice.LetterID)
				messagesPublished++
			} else {
				fmt.Printf("%s: Published Failed Error - LetterID: %d\r\n", time.Now(), notice.LetterID)
				messagesFailedToPublish++
			}
		case err := <-ChannelPool.Errors():
			fmt.Printf("%s: ChannelPool Error - %s\r\n", time.Now(), err)
			channelPoolErrors++
		case err := <-ConnectionPool.Errors():
			fmt.Printf("%s: ConnectionPool Error - %s\r\n", time.Now(), err)
			connectionPoolErrors++
		case err := <-consumer.Errors():
			fmt.Printf("%s: Consumer Error - %s\r\n", time.Now(), err)
			consumerErrors++
		case message := <-consumer.Messages():
			messagesReceived++
			fmt.Printf("%s: ConsumedMessage\r\n", time.Now())
			go func(msg *models.Message) {
				err := msg.Acknowledge()
				if err != nil {
					fmt.Printf("%s: AckMessage Error - %s\r\n", time.Now(), err)
					messagesFailedToAck++
				} else {
					fmt.Printf("%s: AckMessaged\r\n", time.Now())
					messagesAcked++
				}
			}(message)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	fmt.Printf("ChannelPool Errors: %d\r\n", channelPoolErrors)
	fmt.Printf("ConnectionPool Errors: %d\r\n", connectionPoolErrors)

	fmt.Printf("Consumer Errors: %d\r\n", consumerErrors)
	fmt.Printf("Messages Acked: %d\r\n", messagesAcked)
	fmt.Printf("Messages Failed to Ack: %d\r\n", messagesFailedToAck)
	fmt.Printf("Messages Received: %d\r\n", messagesReceived)

	fmt.Printf("Messages Published: %d\r\n", messagesPublished)
	fmt.Printf("Messages Failed to Publish: %d\r\n", messagesFailedToPublish)

	publisher.StopAutoPublish()

	consumer.StopConsuming(false, true)

	ChannelPool.Shutdown()
}

package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"go.uber.org/zap"
	"time"
)

type KafkaQueue struct {
	opts    Options
	log     *zap.SugaredLogger
	w       *kafka.Writer
	readers []*kafka.Reader
	c       *kafka.Client
}

type Options struct {
	Topic     string
	Address   []string
	Mechanism string
	Username  string
	Password  string

	ConsumerNumbers int
}

func (o Options) MechanismX() sasl.Mechanism {
	if true {
		return nil
	}
	mechanism, err := scram.Mechanism(scram.SHA512, "username", "password")
	if err != nil {
		panic(err)
	}
	return mechanism
}

func NewKafkaQueue(opts Options, logger *zap.SugaredLogger) (queue.Queue, error) {

	//mechanism := opts.MechanismX()

	// Transports are responsible for managing connection pools and other resources,
	// it's generally best to create a few of these and share them across your
	// application.
	//sharedTransport := &kafka.Transport{
	//	SASL: mechanism,
	//}

	//kafka.DefaultTransport
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(opts.Address...),
		Topic:                  opts.Topic,
		AllowAutoTopicCreation: true,
		//Balancer:               &kafka.LeastBytes{}, // TODO
		//Transport: sharedTransport,
		//RequiredAcks: kafka.RequireOne,
		Async:       true,
		Compression: kafka.Snappy,
	}

	readers := make([]*kafka.Reader, 0, opts.ConsumerNumbers)
	for i := 0; i < opts.ConsumerNumbers; i++ {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  opts.Address,
			Topic:    opts.Topic,
			GroupID:  "webhookx",
			MaxBytes: 10e6, // 10MB
			//Dialer: &kafka.Dialer{
			//	Timeout:       kafka.DefaultDialer.Timeout,
			//	DualStack:     kafka.DefaultDialer.DualStack,
			//	SASLMechanism: mechanism,
			//},
			MaxWait: 10 * time.Second,
			//CommitInterval:
		})
		readers = append(readers, reader)
	}

	client := &kafka.Client{
		Addr: kafka.TCP(opts.Address...),
	}

	q := &KafkaQueue{
		opts:    opts,
		log:     logger,
		w:       writer,
		readers: readers,
		c:       client,
	}

	return q, nil
}

func (q *KafkaQueue) Enqueue(ctx context.Context, message *queue.Message) error {
	m := kafka.Message{
		Key:   nil, // TODO
		Value: message.Value,
		Headers: []kafka.Header{
			{Key: "wid", Value: []byte(message.WorkspaceID)},
		},
	}
	return q.w.WriteMessages(ctx, m)
}

func (q *KafkaQueue) Size(ctx context.Context) (int64, error) {
	return 0, nil
}

func (q *KafkaQueue) Stats() map[string]interface{} {
	return map[string]interface{}{}
}

func (q *KafkaQueue) Close() error {
	if err := q.w.Close(); err != nil {
		return err
	}
	for i := range q.readers {
		if err := q.readers[i].Close(); err != nil {
			return err
		}

	}
	return nil
}

func fetchMessages(ctx context.Context, r *kafka.Reader) ([]kafka.Message, error) {
	list := make([]kafka.Message, 0)
	for i := 0; i < 20; i++ {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			fmt.Println("failed to fetch message")
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (q *KafkaQueue) StartListen(ctx context.Context, handle queue.HandlerFunc) {
	for i := 0; i < q.opts.ConsumerNumbers; i++ {
		go q.listen(ctx, q.readers[i], handle)
	}
}

func toMessage(message kafka.Message) *queue.Message {
	return &queue.Message{
		Value:       message.Value,
		WorkspaceID: string(message.Headers[0].Value),
	}
}

func (q *KafkaQueue) listen(ctx context.Context, reader *kafka.Reader, handle queue.HandlerFunc) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				xmessages, err := fetchMessages(ctx, reader)
				if err != nil {
					fmt.Println("failed to fetch message", err)
					continue
				}

				if len(xmessages) == 0 {
					continue
				}

				messages := make([]*queue.Message, 0, len(xmessages))
				for _, msg := range xmessages {
					messages = append(messages, toMessage(msg))
				}

				err = handle(ctx, messages)
				if err != nil {
					q.log.Warnf("failed to handle message: %v", err)
					continue
				}
				err = reader.CommitMessages(ctx, xmessages...)
				if err != nil {
					q.log.Warnf("failed to delete message: %v", err)
				}
			}
		}
	}()
}

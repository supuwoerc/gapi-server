package email

import (
	"context"
	"crypto/tls"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, msg *Message) error
	SendBatch(ctx context.Context, msgs []*Message) error
}

type Client struct {
	dialer *gomail.Dialer
	from   string
	name   string
	logger *logger.Logger
}

func NewClient(cfg *config.EmailConfig, l *logger.Logger) *Client {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if cfg.UseTLS {
		d.TLSConfig = &tls.Config{ServerName: cfg.Host}
	} else {
		d.SSL = false
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		dialer: d,
		from:   cfg.FromAddress,
		name:   cfg.FromName,
		logger: l,
	}
}

func (c *Client) Send(ctx context.Context, msg *Message) error {
	m := c.buildMessage(msg)
	if err := c.dialer.DialAndSend(m); err != nil {
		c.logger.Ctx(ctx).Error("failed to send email", zap.Strings("to", msg.To), zap.Error(err))
		return err
	}
	return nil
}

func (c *Client) SendBatch(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}
	sender, err := c.dialer.Dial()
	if err != nil {
		c.logger.Ctx(ctx).Error("failed to dial SMTP server", zap.Error(err))
		return err
	}
	defer func() {
		if err := sender.Close(); err != nil {
			c.logger.Ctx(ctx).Error("failed to close SMTP connection", zap.Error(err))
		}
	}()
	for _, msg := range msgs {
		m := c.buildMessage(msg)
		if err := gomail.Send(sender, m); err != nil {
			c.logger.Ctx(ctx).Error("failed to send email in batch", zap.Strings("to", msg.To), zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Client) buildMessage(msg *Message) *gomail.Message {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(c.from, c.name))
	m.SetHeader("To", msg.To...)
	m.SetHeader("Subject", msg.Subject)
	if msg.HTML != "" {
		m.SetBody("text/html", msg.HTML)
		if msg.Text != "" {
			m.AddAlternative("text/plain", msg.Text)
		}
	} else {
		m.SetBody("text/plain", msg.Text)
	}
	return m
}

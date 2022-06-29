package client

import (
	"context"
	"fmt"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"strconv"
)

type Client struct {
	client       *resty.Client
	serverConfig *config.ServerConfig
	log          *zerolog.Logger
}

func InitClient(serverConfig *config.ServerConfig, log *zerolog.Logger) *Client {
	accrualClient := resty.New()
	log.Info().Msg("accrual service client initialized")
	return &Client{client: accrualClient, serverConfig: serverConfig, log: log}
}

func (c *Client) GetAccrual(ctx context.Context, orderNumber int) (*resty.Response, error) {
	log.Info().Msg(fmt.Sprintf("sending request for order %v", orderNumber))
	response, err := c.client.R().SetContext(ctx).SetPathParams(map[string]string{"orderNumber": strconv.Itoa(orderNumber)}).Get(c.serverConfig.AccrualAddress + "/api/orders/{orderNumber}")
	if err != nil {
		c.log.Err(err).Msg(fmt.Sprintf("accrual retrieval from service failed for order %v", orderNumber))
		return nil, err
	}
	return response, nil
}
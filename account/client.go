package account

import (
	"context"

	"github.com/sdshah09/GoCore/account/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	service pb.AccountServiceClient
}

func NewClient(url string) (*Client, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	service := pb.NewAccountServiceClient(conn)
	return &Client{
		conn:    conn,
		service: service,
	}, nil
}

func (client *Client) Close() {
	client.conn.Close()
}

func (client *Client) PostAccount(ctx context.Context, name string) (*Account, error) {
	res, err := client.service.PostAccount(
		ctx,
		&pb.PostAccountRequest{Name: name},
	)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:   res.Account.Id,
		Name: res.Account.Name,
	}, nil
}

func (client *Client) GetAccount(ctx context.Context, id string) (*Account, error) {
	res, err := client.service.GetAccount(
		ctx,
		&pb.GetAccountRequest{Id: id},
	)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:   res.Account.Id,
		Name: res.Account.Name,
	}, nil
}

func (client *Client) GetAccounts(ctx context.Context, skip uint64, take uint64) ([]Account, error) {
	res, err := client.service.GetAccounts(
		ctx,
		&pb.GetAccountsRequest{Skip: skip, Take: take},
	)
	if err != nil {
		return nil, err
	}
	var accounts []Account
	for _, pbAccount := range res.Accounts {
		accounts = append(accounts, Account{
			ID:   pbAccount.Id,
			Name: pbAccount.Name,
		})
	}

	return accounts, nil
}

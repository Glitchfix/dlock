package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Glitchfix/dlock/client"
	"github.com/Glitchfix/dlock/lock"
	"github.com/Glitchfix/dlock/proto/plugin"
	"google.golang.org/grpc"
)

func server(lis net.Listener) {

	// Create a new gRPC server
	s := grpc.NewServer()

	// Register the Greeter service with the server
	plugin.RegisterDLockServer(s, lock.NewServer())

	log.Info().Msgf("lock server started")

	// Start the server
	if err := s.Serve(lis); err != nil {
		log.Error().Msgf("server shutdown: %v\n", err)
	}
}

func main() {
	arr := []int{}
	count := 100

	for i := 0; i < count; i++ {
		arr = append(arr, i)
	}

	wg := &sync.WaitGroup{}

	// wr, _ := os.OpenFile("output.log", os.O_RDWR, 0666)
	n := 0

	// Create a TCP listener on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Info().Msgf("Failed to listen: %v\n", err)
		return
	}
	defer lis.Close()
	go server(lis)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func(arr *[]int, wg *sync.WaitGroup, i int) {
			defer wg.Done()
			// Set up a connection to the gRPC server.
			conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
			if err != nil {
				log.Fatal().Msgf("Failed to connect: %v", err)
			}
			defer conn.Close()

			// Register reflection service on gRPC server
			lk := client.NewClient(conn)

			id := fmt.Sprintf("acquirer%d", i)

			// log.Info().Msgf("[%d] %s trying to acquire LOCK\n", n, id)
			lk.Lock("arr", id)
			log.Debug().Msgf("[%d] %s acquired LOCK\n", n, id)
			defer lk.Unlock("arr", id)
			log.Debug().Msgf("[%d] %s will UNLOCK\n", n, id)

			N := len(*arr)

			last := (*arr)[N-1]
			*arr = (*arr)[:N-1]
			n += 1
			if N-2 < 0 {
				log.Info().Msgf("[%d] %s print: %d %v\n", n, id, last, *arr)
				return
			}

			log.Info().Msgf("[%d] %s print: %d %v\n", n, id, last, (*arr)[N-2])

		}(&arr, wg, i)
	}
	wg.Wait()
}

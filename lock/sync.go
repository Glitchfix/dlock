package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Glitchfix/dlock/proto/plugin"
	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
)

const (
	daemonSetName = "dlock-daemonset"
)

func (l *Server) Boot() {
	// Create the Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(uuid.New().String()) // Replace with appropriate node ID

	// Create the snapshot store
	snapshots, err := raft.NewFileSnapshotStore("raft-snapshots", 1, nil)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Create the log store
	logStore := raft.NewInmemStore()

	// Create the stable store
	stableStore := raft.NewInmemStore()

	// Create the transport layer
	addr := "127.0.0.1:6000" // Replace with appropriate address
	transport, err := raft.NewTCPTransport(addr, nil, 3, 10*time.Second, nil)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Create the Raft instance
	l.raftInstance, err = raft.NewRaft(config, nil, logStore, stableStore, snapshots, transport)
	if err != nil {
		log.Fatal().Err(err)
	}

	l.blankSink = &LockServerBlankSink{}

}

func (l *Server) Announce() {
	// use the current context in kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Get the list of pods created by the DaemonSet
	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("dlock/app=%s", daemonSetName),
	})
	if err != nil {
		log.Fatal().Err(err)
	}

	for _, pod := range podList.Items {
		log.Debug().Msgf("Pod Name: %s\n", pod.Name)
		podIP := pod.Status.PodIP

	}
}

func (l *Server) Sync() {
	for {
		select {
		case isLeader := <-l.raftInstance.LeaderCh():
			// Logic for handling leadership events
			if !isLeader {
				log.Debug().Msgf("Leadership lost by %s.", l.ID)
			} else {
				log.Debug().Msgf("Leadership acquired by %s.", l.ID)
			}
		}
	}

}

func (l *Server) LeadershipChanged() {
	// Get the current Raft state using State() method
	currentState := l.raftInstance.State()
	// Handle the current state as needed
	switch currentState {
	case raft.Follower:
		log.Info().Msgf("node %s is now a follower.", l.ID)
	case raft.Candidate:
		log.Info().Msgf("node %s is now a candidate.", l.ID)
	case raft.Leader:
		log.Info().Msgf("node %s is now the leader.", l.ID)
	case raft.Shutdown:
		log.Info().Msgf("node %s is now shuting down.", l.ID)
	}
}

type LockServerBlankSink struct{}

func (l *LockServerBlankSink) Send(lreq *plugin.LockResponse) (err error) {
	return
}

func (l *Server) SetHeader(metadata.MD) error {
	return errors.New("NOT IMPLEMENTED")
}
func (l *Server) SendHeader(metadata.MD) error {
	return errors.New("NOT IMPLEMENTED")
}
func (l *Server) SetTrailer(metadata.MD) {
}
func (l *Server) SendMsg(m any) error {
	return errors.New("NOT IMPLEMENTED")
}
func (l *Server) RecvMsg(m any) error {
	return errors.New("NOT IMPLEMENTED")
}

type name struct {
}

func (l *Server) Apply(log *raft.Log) interface{} {
	switch log.Type {
	case raft.LogCommand:
		lreq := &plugin.LockRequest{}
		if err := proto.Unmarshal(log.Data, lreq); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error marshalling store payload %s\n", err.Error())
			return nil
		}

		switch lreq.Operation {
		case plugin.OperationType_Lock:
			go l.Lock(lreq, nil)
		case plugin.OperationType_Unlock:
			go l.Unlock(lreq, nil)
		}
	}
	return nil
}

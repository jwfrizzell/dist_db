package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dist_db/cmd"
	"github.com/gorilla/mux"
	"github.com/hashicorp/serf/serf"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func main() {
	a := resolveIP()
	if len(a) == 0 {
		log.Fatal("Unable to resolve IP address")
		os.Exit(1)
	}
	c := os.Getenv("CLUSTER_ADDR")

	cluster, err := setupCluster(a, c)
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Leave()
	theOneAndOnlyNumber := cmd.InitTheNumber(42)
	launchHTTPAPI(theOneAndOnlyNumber)

	ctx := context.Background()
	if name, err := os.Hostname(); err == nil {
		ctx = context.WithValue(ctx, "name", name)
	}

	debugDataPrinterTicker := time.Tick(time.Second * 5)
	numberBroadcastTicker := time.Tick(time.Second * 2)

	for {
		select {
		case <-numberBroadcastTicker:
			members := getOtherMembers(cluster)

			ctx, _ := context.WithTimeout(ctx, time.Second*2)
			go notifyOthers(ctx, members, theOneAndOnlyNumber)
		case <-debugDataPrinterTicker:
			log.Printf("Members: %v\n", cluster.Members())

			curVal, curGen := theOneAndOnlyNumber.GetValue()
			log.Printf("State: Val: %v Gen: %v\n", curVal, curGen)
		}
	}
}

func setupCluster(advertiseAddr string, clusterAddr string) (*serf.Serf, error) {
	conf := serf.DefaultConfig()
	conf.Init()
	conf.MemberlistConfig.AdvertiseAddr = advertiseAddr
	cluster, err := serf.Create(conf)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't create cluster")
	}

	_, err = cluster.Join([]string{clusterAddr}, true)
	if err != nil {
		log.Printf("Couldn't join cluster, start own: %v\n", err)
	}
	return cluster, nil
}

func launchHTTPAPI(db *cmd.OneAndOnlyNumber) {
	go func() {
		m := mux.NewRouter()

		m.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
			val, _ := db.GetValue()
			fmt.Fprintf(w, "%v", val)
		})

		m.HandleFunc("/set/{newVal}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			newVal, err := strconv.Atoi(vars["newVal"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "%v", err)
			}
			db.SetValue(newVal)
			fmt.Fprintf(w, "%v", newVal)
		})

		m.HandleFunc("/notify/{curVal}/{curGeneration}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			curVal, err := strconv.Atoi(vars["curVal"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "%v", err)
				return
			}

			curGeneration, err := strconv.Atoi(vars["curGeneration"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "%v", err)
				return
			}

			if changed := db.NotifyValue(curVal, curGeneration); changed {
				log.Printf("NewVal: %v Gen: %v Notifier: %v",
					curVal,
					curGeneration,
					r.URL.Query().Get("notifier"))
			}
			w.WriteHeader(http.StatusOK)
		})
		log.Fatal(http.ListenAndServe(":8080", m))
	}()
}

func getOtherMembers(cluster *serf.Serf) []serf.Member {
	members := cluster.Members()
	for i := 0; i < len(members); {
		if members[i].Name == cluster.LocalMember().Name ||
			members[i].Status != serf.StatusAlive {
			if i < len(members)-1 {
				members = append(members[:i], members[i+1:]...)
			} else {
				members = members[:1]
			}
		} else {
			i++
		}
	}
	return members
}

func notifyOthers(ctx context.Context, otherMembers []serf.Member, db *cmd.OneAndOnlyNumber) {
	g, ctx := errgroup.WithContext(ctx)

	if len(otherMembers) <= cmd.MembersToNotify {
		for _, member := range otherMembers {
			curMember := member
			g.Go(func() error {
				return notifyMember(ctx, curMember.Addr.String(), db)
			})
		}
	} else {
		randIndex := rand.Int() % len(otherMembers)
		for i := 0; i < cmd.MembersToNotify; i++ {
			g.Go(func() error {
				return notifyMember(ctx, otherMembers[(randIndex+i)%len(otherMembers)].Addr.String(), db)
			})
		}
	}

	err := g.Wait()
	if err != nil {
		log.Printf("Error when notifying other members: %v", err)
	}
}

func notifyMember(ctx context.Context, addr string, db *cmd.OneAndOnlyNumber) error {
	val, gen := db.GetValue()
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v:8080/notify/%v/%v?notifier=%v",
		addr, val, gen, ctx.Value("name")), nil)
	if err != nil {
		return errors.Wrap(err, "Couldn't create request")
	}
	req = req.WithContext(ctx)

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Couldn't make request")
	}
	return nil
}

func resolveIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("resolveIP Error: %s", err)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("IP", ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

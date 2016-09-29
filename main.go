package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"
	"github.com/stripe/stripe-go/sub"
)

// Sorting
type Subs []*stripe.Sub

func (slice Subs) Len() int {
	return len(slice)
}
func (slice Subs) Less(i, j int) bool {
	return slice[i].PeriodEnd < slice[j].PeriodEnd
}
func (slice Subs) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func combine(ids []string, key string, run bool) error {
	if len(ids) < 2 {
		return fmt.Errorf("need at least 2 subscription IDs, got %d", len(ids))
	}

	if len(ids) > 20 {
		return fmt.Errorf("a maximum of 20 subscriptions can be combined, got %d", len(ids))
	}

	if run {
		log.Println("Running in real mode; subscriptions will be changed.")
	} else {
		log.Println("Running in dry-run mode; no changes will be made.")
	}

	api := client.New(key, nil)

	// Fetch the requested subscriptions
	subs := []*stripe.Sub{}
	for _, id := range ids {
		s, err := api.Subs.Get(id, nil)
		if err != nil {
			return err
		}
		subs = append(subs, s)
	}

	// Filter out canceled subscriptions
	active := []*stripe.Sub{}
	for _, s := range subs {
		if s.Status == sub.Canceled {
			log.Printf("subscription %s is canceled, ignoring.", s.ID)
			continue
		}
		active = append(active, s)
	}

	if len(active) == 0 {
		return fmt.Errorf("All provided subscriptions are canceled.")
	}

	if len(active) == 1 {
		log.Printf("Only one subscription (%s) is active, migration complete.\n", active[0].ID)
		return nil
	}

	// Sort the subsriptions by current period end
	var sorted Subs
	sorted = active
	sort.Sort(sorted)

	primary, rest := sorted[len(sorted)-1], sorted[:len(sorted)-1]

	log.Printf("Using subscription %s as the primary subscription", primary.ID)

	// Verify that all subscriptions share the same billing interval
	sharedPlan := primary.Items.Values[0].Plan
	for _, s := range active {
		plan := s.Items.Values[0].Plan
		if plan.Interval != sharedPlan.Interval {
			return fmt.Errorf("Subscription %s bills %sly, not %sly", plan.Interval, sharedPlan.Interval)
		}
		if plan.IntervalCount != sharedPlan.IntervalCount {
			return fmt.Errorf("Subscription %s bills every %d %s, not every %d %s", plan.IntervalCount, sharedPlan.IntervalCount)
		}
	}

	// Updating the subscription
	items := []*stripe.SubItemParams{}

	for _, s := range rest {
		items = append(items, &stripe.SubItemParams{
			Plan:     s.Plan.ID,
			Quantity: s.Quantity,
		})
	}

	log.Printf("Adding the following items to the primary subscription:")
	for _, item := range items {
		log.Printf("- Plan: %s, Quantity: %d\n", item.Plan, item.Quantity)
	}

	if len(items)+1 == len(primary.Items.Values) {
		log.Printf("Primary subscription has already been updated with the correct number of items")
	} else if run {
		_, err := api.Subs.Update(primary.ID, &stripe.SubParams{
			Items: items,
		})
		if err != nil {
			return err
		}
	}

	// Canceling the rest
	for _, s := range rest {
		log.Printf("Canceling subscription %s", s.ID)
		if _, err := api.Subs.Cancel(s.ID, nil); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	var run = flag.Bool("run", false, "combine the subscriptions")
	var key = flag.String("key", "", "Stripe API key")
	flag.Parse()
	if err := combine(flag.Args(), *key, *run); err != nil {
		log.Fatal(err)
	}
}

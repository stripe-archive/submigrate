package main

import (
	"flag"
	"fmt"
	"os"
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

func mark(f string, args ...interface{}) {
	fmt.Printf("-----> " + fmt.Sprintf(f, args...) + "\n")
}

func errf(err error) {
	fmt.Println()
	fmt.Printf("ERROR! Migration unsuccessful\n")
	log(err.Error())
}

func log(f string, args ...interface{}) {
	fmt.Printf("       " + fmt.Sprintf(f, args...) + "\n")
}

func combine(ids []string, key string, run bool) error {
	if len(ids) < 2 {
		return fmt.Errorf("At least two subscription IDs are neeed, got %d", len(ids))
	}

	if len(ids) > 20 {
		return fmt.Errorf("A maximum of 20 subscriptions can be combined, got %d", len(ids))
	}

	if run {
		mark("Begginning subscription migration in REAL mode, subscriptions will be changed")
	} else {
		mark("Begginning subscription migration in DRYRUN mode, no changes will be made")
	}
	log("")

	api := client.New(key, nil)

	mark("Fetching requested subscriptions")
	subs := []*stripe.Sub{}
	for _, id := range ids {
		s, err := api.Subs.Get(id, nil)
		log("Getting subscription %s", s.ID)
		if err != nil {
			return err
		}
		subs = append(subs, s)
	}
	log("Successfully fetched subscriptions\n")

	mark("Filtering out canceled subscriptions")
	active := []*stripe.Sub{}
	for _, s := range subs {
		if s.Status == sub.Canceled {
			log("Ignoring canceled subscription %s", s.ID)
			continue
		}
		active = append(active, s)
	}
	log("")

	if len(active) == 0 {
		return fmt.Errorf("All provided subscriptions are canceled")
	}

	if len(active) == 1 {
		mark("Marking migration as complete")
		log("Subscription %s is active and all other subscriptions are canceled", active[0].ID)
		return nil
	}

	// Sort the subsriptions by current period end
	var sorted Subs
	sorted = active
	sort.Sort(sorted)

	primary, rest := sorted[len(sorted)-1], sorted[:len(sorted)-1]

	mark("Migrating subscriptions")
	log("Using subscription %s as the primary subscription", primary.ID)
	for _, item := range primary.Items.Values {
		log("> Plan: %s (%s), Quantity: %d", item.Plan.Name, item.Plan.ID, item.Quantity)
	}
	log("")

	// Verify that all subscriptions share the same billing interval
	errors := false
	sharedPlan := primary.Items.Values[0].Plan
	sharedCustomer := primary.Customer
	for _, s := range active {
		plan := s.Items.Values[0].Plan
		if s.Customer.ID != sharedCustomer.ID {
			log("! Mismatch on subscription %s: Subscription is for customer %s, not %s", s.ID, s.Customer.ID, sharedCustomer.ID)
			errors = true
		}
		if plan.Currency != sharedPlan.Currency {
			log("! Mismatch on subscription %s: Plan %s bills in %s, not %s", s.ID, plan.ID, plan.Currency, sharedPlan.Currency)
			errors = true
		}
		if plan.Interval != sharedPlan.Interval {
			log("! Mismatch on subscription %s: Plan %s bills %sly, not %sly", s.ID, plan.ID, plan.Interval, sharedPlan.Interval)
			errors = true
		}
		if plan.TrialPeriod > 0 {
			log("! Mismatch on subscription %s: Plan %s has a trial period", s.ID, plan.ID)
			errors = true
		}
		if plan.IntervalCount != sharedPlan.IntervalCount {
			log("! Mismatch on subscription %s: Plan %s bills every %d %ss, not every %d %ss", s.ID, plan.ID, plan.IntervalCount, plan.Interval, sharedPlan.IntervalCount, sharedPlan.Interval)
			errors = true
		}
	}

	if errors {
		return fmt.Errorf("Subscriptions and related plans have properties that do not match")
	}

	count := 0
	log("Adding the following items to the primary subscription")
	for _, s := range rest {
		for _, item := range s.Items.Values {
			log("> Plan: %s (%s), Quantity: %d", item.Plan.Name, item.Plan.ID, item.Quantity)
			count += 1
		}
	}
	log("")

	if count+1 == len(primary.Items.Values) {
		log("Previously updated subscription, skipping")
	} else {
		log("Updating primary subscription, prorating other subscriptions")
		for _, s := range rest {
			items := []*stripe.SubItemsParams{}
			for _, item := range s.Items.Values {
				log("Added plan %s with quantity %d to %s", item.Plan.ID, item.Quantity, primary.ID)
				items = append(items, &stripe.SubItemsParams{
					Plan:     item.Plan.ID,
					Quantity: item.Quantity,
				})
			}
			if run {
				_, err := api.Subs.Update(primary.ID, &stripe.SubParams{
					Items:         items,
					ProrationDate: s.PeriodEnd,
				})
				if err != nil {
					return err
				}
			}
		}
		log("Successfully updated primary subscription")
	}
	log("")

	mark("Canceling remaining subscriptions")
	for _, s := range rest {
		log("Ending subscription %s", s.ID)
		if run {
			if _, err := api.Subs.Cancel(s.ID, nil); err != nil {
				return err
			}
		}
	}
	log("Successfully completed migration")

	return nil
}

func main() {
	// Turn off Stripe logging
	stripe.LogLevel = 0

	var run = flag.Bool("run", false, "Run the migration; by default no actions are taken")
	var key = flag.String("key", "", "Stripe API key")
	flag.Parse()
	if err := combine(flag.Args(), *key, *run); err != nil {
		errf(err)
		os.Exit(1)
	}
}

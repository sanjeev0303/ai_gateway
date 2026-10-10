package main
import (
	"fmt"
	"os"
	"strings"
)
func main() {
	b, _ := os.ReadFile("backend/internal/bootstrap/run.go")
	s := string(b)
	
	// Gateway InProcess patch
	s = strings.Replace(s,
		`			jobs.Add(1)
			go func() {
				defer jobs.Done()
				worker.RunUsageConsumer(ctx, bus.EventBus, usageSvc, log)
			}()`,
		`			secSvc := securityevent.NewService(securityevent.ServiceDeps{Events: repositories.NewSecurityEventRepository(store)})
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				worker.RunUsageConsumer(ctx, bus.EventBus, usageSvc, log)
			}()
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				worker.RunSecurityConsumer(ctx, bus.EventBus, secSvc, log)
			}()`, 1)
			
	// Worker patch
	s = strings.Replace(s,
		`			if !bus.inProcess {
				jobs.Add(1)
				go func() {
					defer jobs.Done()
					worker.RunUsageConsumer(ctx, bus.EventBus, rollups, log)
				}()
			}`,
		`			if !bus.inProcess {
				secSvc := securityevent.NewService(securityevent.ServiceDeps{Events: repositories.NewSecurityEventRepository(store)})
				jobs.Add(1)
				go func() {
					defer jobs.Done()
					worker.RunUsageConsumer(ctx, bus.EventBus, rollups, log)
				}()
				jobs.Add(1)
				go func() {
					defer jobs.Done()
					worker.RunSecurityConsumer(ctx, bus.EventBus, secSvc, log)
				}()
			}`, 1)

	// Add securityevent import if not there
	if !strings.Contains(s, `"ai-governance-saas/backend/internal/application/securityevent"`) {
		s = strings.Replace(s, `"ai-governance-saas/backend/internal/application/usage"`, `"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/application/securityevent"`, 1)
	}

	os.WriteFile("backend/internal/bootstrap/run.go", []byte(s), 0644)
	fmt.Println("run.go patched")
}

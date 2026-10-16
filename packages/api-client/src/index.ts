import type { ApiClientOptions, Transport } from "./client";
import { createTransport } from "./client";
import type { ActivityModule } from "./activity";
import { createActivityModule } from "./activity";
import type { AuthModule } from "./auth";
import { createAuthModule } from "./auth";
import type { AlertsModule } from "./alerts";
import { createAlertsModule } from "./alerts";
import type { AuditLogsModule } from "./audit-logs";
import { createAuditLogsModule } from "./audit-logs";
import type { BillingModule } from "./billing";
import { createBillingModule } from "./billing";
import type { ConversationsModule } from "./conversations";
import { createConversationsModule } from "./conversations";
import type { CredentialsModule } from "./credentials";
import { createCredentialsModule } from "./credentials";
import type { InvitationsModule } from "./invitations";
import { createInvitationsModule } from "./invitations";
import type { MembersModule } from "./members";
import { createMembersModule } from "./members";
import type { PoliciesModule } from "./policies";
import { createPoliciesModule } from "./policies";
import type { ModelAccessModule, BudgetsModule } from "./model_access_budget";
import { createModelAccessModule, createBudgetsModule } from "./model_access_budget";
import type { OrganizationModule } from "./organization";
import { createOrganizationModule } from "./organization";
import type { ModelsModule } from "./models";
import { createModelsModule } from "./models";
import type { ProjectsModule } from "./projects";
import { createProjectsModule } from "./projects";
import type { ProvidersModule } from "./providers";
import { createProvidersModule } from "./providers";
import type { SecurityEventsModule } from "./security-events";
import { createSecurityEventsModule } from "./security-events";
import type { ServiceAccountsModule } from "./service-accounts";
import { createServiceAccountsModule } from "./service-accounts";
import type { TeamsModule } from "./teams";
import { createTeamsModule } from "./teams";
import type { UsageModule } from "./usage";
import { createUsageModule } from "./usage";
import type { SearchModule } from "./search";
import { createSearchModule } from "./search";
import type { GuardrailsModule } from "./guardrails";
import { createGuardrailsModule } from "./guardrails";

export * from "./errors";
export { createTransport, paginate } from "./client";
export type {
  ApiClientOptions,
  ContractPath,
  FetchLike,
  HttpMethod,
  ListResponse,
  Page,
  Query,
  QueryValue,
  RequestOptions,
  Transport,
} from "./client";

export * from "./activity";
export * from "./auth";
export * from "./alerts";
export * from "./audit-logs";
export * from "./billing";
export * from "./conversations";
export * from "./credentials";
export * from "./guardrails";
export * from "./invitations";
export * from "./members";
export * from "./policies";
export * from "./model_access_budget";
export * from "./organization";
export * from "./models";
export * from "./projects";
export * from "./providers";
export * from "./security-events";
export * from "./service-accounts";
export * from "./teams";
export * from "./usage";
export * from "./search";

/**
 * Transport plus one module per API area. Module names follow the REST nouns,
 * except `serviceAccounts` (camelCase) and `credentials`, which covers both the
 * control-plane and gateway credential routes.
 */
export interface ApiClient extends Transport {
  readonly auth: AuthModule;
  readonly policies: PoliciesModule;
  readonly modelAccess: ModelAccessModule;
  readonly budgets: BudgetsModule;
  readonly guardrails: GuardrailsModule;
  readonly organization: OrganizationModule;
  readonly members: MembersModule;
  readonly invitations: InvitationsModule;
  readonly projects: ProjectsModule;
  readonly teams: TeamsModule;
  readonly serviceAccounts: ServiceAccountsModule;
  readonly credentials: CredentialsModule;
  readonly providers: ProvidersModule;
  readonly models: ModelsModule;
  readonly usage: UsageModule;
  readonly conversations: ConversationsModule;
  readonly auditLogs: AuditLogsModule;
  readonly billing: BillingModule;
  readonly securityEvents: SecurityEventsModule;
  readonly alerts: AlertsModule;
  readonly activity: ActivityModule;
  readonly search: SearchModule;
}

export function createApiClient(options: ApiClientOptions): ApiClient {
  const transport = createTransport(options);
  return {
    ...transport,
    auth: createAuthModule(transport),
    policies: createPoliciesModule(transport),
    modelAccess: createModelAccessModule(transport),
    budgets: createBudgetsModule(transport),
    guardrails: createGuardrailsModule(transport),
    organization: createOrganizationModule(transport),
    members: createMembersModule(transport),
    invitations: createInvitationsModule(transport),
    projects: createProjectsModule(transport),
    teams: createTeamsModule(transport),
    serviceAccounts: createServiceAccountsModule(transport),
    credentials: createCredentialsModule(transport),
    providers: createProvidersModule(transport),
    models: createModelsModule(transport),
    usage: createUsageModule(transport),
    conversations: createConversationsModule(transport),
    auditLogs: createAuditLogsModule(transport),
    billing: createBillingModule(transport),
    securityEvents: createSecurityEventsModule(transport),
    alerts: createAlertsModule(transport),
    activity: createActivityModule(transport),
    search: createSearchModule(transport),
  };
}

export * from "./gateway";
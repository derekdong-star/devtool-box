package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"devtoolbox/internal/model"
)

const (
	tokenRouterReadConnName     = "tokenrouter-read"
	businessQuerySectionTimeout = 30 * time.Second
	defaultRecentSessionLimit   = 20
	maxRecentSessionLimit       = 100
)

type BusinessQueryService struct {
	connStore *ConnStore
	db        *DBService
	location  *time.Location
}

func NewBusinessQueryService(connStore *ConnStore) *BusinessQueryService {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return &BusinessQueryService{
		connStore: connStore,
		db:        NewDBService(),
		location:  location,
	}
}

func (s *BusinessQueryService) TokenRouterUserOverview(ctx context.Context, req model.BusinessQueryReq) (model.BusinessQueryResp, error) {
	identifierType, identifier, err := normalizeBusinessIdentifier(req.Identifier)
	if err != nil {
		return model.BusinessQueryResp{}, err
	}

	conn, err := s.tokenRouterReadConn()
	if err != nil {
		return model.BusinessQueryResp{}, err
	}

	sections := make([]model.BusinessQuerySection, 0, 5)
	for _, spec := range tokenRouterOverviewQueries(identifierType) {
		section, err := s.querySection(ctx, conn, spec, identifier, req)
		if err != nil {
			return model.BusinessQueryResp{}, fmt.Errorf("%s 查询失败: %w", spec.title, err)
		}
		sections = append(sections, section)
	}

	return model.BusinessQueryResp{
		IdentifierType:       identifierType,
		NormalizedIdentifier: identifier,
		Sections:             sections,
	}, nil
}

func (s *BusinessQueryService) tokenRouterReadConn() (model.DBConn, error) {
	conn, ok, err := s.connStore.FindByName(tokenRouterReadConnName)
	if err != nil {
		return model.DBConn{}, err
	}
	if !ok {
		return model.DBConn{}, fmt.Errorf("未找到数据库连接 %q，请先在数据库查询模块保存该 PostgreSQL 只读连接", tokenRouterReadConnName)
	}
	if conn.Type != "postgres" {
		return model.DBConn{}, fmt.Errorf("数据库连接 %q 的类型是 %q，需要 PostgreSQL", tokenRouterReadConnName, conn.Type)
	}
	return conn, nil
}

func (s *BusinessQueryService) querySection(ctx context.Context, conn model.DBConn, spec businessQuerySpec, identifier string, req model.BusinessQueryReq) (model.BusinessQuerySection, error) {
	queryCtx, cancel := context.WithTimeout(ctx, businessQuerySectionTimeout)
	defer cancel()

	args := []interface{}{identifier}
	if spec.key == "sessions" {
		args = append(args, req.ActiveSessionsOnly, clampSessionLimit(req.RecentSessionLimit))
	}
	columns, rows, err := s.db.QueryContext(queryCtx, conn.Type, conn.DSN, spec.sql, args...)
	if err != nil {
		return model.BusinessQuerySection{}, err
	}
	return model.BusinessQuerySection{
		Key:     spec.key,
		Title:   spec.title,
		Columns: columns,
		Rows:    s.formatRows(rows),
	}, nil
}

func (s *BusinessQueryService) formatRows(rows []map[string]interface{}) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		next := make(map[string]interface{}, len(row))
		for key, value := range row {
			next[key] = s.formatValue(value)
		}
		formatted = append(formatted, next)
	}
	return formatted
}

func (s *BusinessQueryService) formatValue(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return v.In(s.location).Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

func normalizeBusinessIdentifier(raw string) (string, string, error) {
	identifier := strings.TrimSpace(raw)
	if identifier == "" {
		return "", "", fmt.Errorf("请输入用户邮箱或用户 UUID")
	}
	if parsed, err := uuid.Parse(identifier); err == nil {
		return "uuid", parsed.String(), nil
	}
	if strings.Contains(identifier, "@") {
		return "email", strings.ToLower(strings.TrimSpace(identifier)), nil
	}
	return "", "", fmt.Errorf("请输入合法的用户邮箱或用户 UUID")
}

func clampSessionLimit(limit int) int {
	if limit <= 0 {
		return defaultRecentSessionLimit
	}
	if limit > maxRecentSessionLimit {
		return maxRecentSessionLimit
	}
	return limit
}

type businessQuerySpec struct {
	key   string
	title string
	sql   string
}

func tokenRouterOverviewQueries(identifierType string) []businessQuerySpec {
	return []businessQuerySpec{
		{key: "identities", title: "用户身份", sql: tokenRouterIdentitySQL(identifierType)},
		{key: "sessions", title: "Session 信息", sql: tokenRouterSessionSQL(identifierType)},
		{key: "wallets", title: "钱包余额", sql: tokenRouterWalletSQL(identifierType)},
		{key: "organizations", title: "组织信息", sql: tokenRouterOrganizationSQL(identifierType)},
		{key: "members", title: "组织成员额度", sql: tokenRouterMemberSQL(identifierType)},
	}
}

func tokenRouterIdentitySQL(identifierType string) string {
	if identifierType == "email" {
		return `
WITH "input" AS (
  SELECT LOWER(BTRIM($1)) AS "email"
)
SELECT
  'PERSONAL'::text AS "accountType",
  "u"."id"::text AS "identityId",
  "u"."id"::text AS "walletUserId",
  NULL::text AS "memberId",
  NULL::text AS "orgId",
  NULL::text AS "orgName",
  NULL::text AS "role",
  "u"."userName" AS "displayName",
  "u"."email" AS "loginName",
  TRUE AS "available",
  ("u"."passwordHash" IS NOT NULL) AS "hasPassword",
  "u"."passwordUpdatedAt",
  "u"."createdAt",
  "u"."updatedAt"
FROM "public"."User" AS "u"
CROSS JOIN "input" AS "i"
WHERE LOWER(BTRIM("u"."email")) = "i"."email"

UNION ALL

SELECT
  'ORG'::text AS "accountType",
  "m"."id"::text AS "identityId",
  "o"."ownerPbdUserId"::text AS "walletUserId",
  "m"."id"::text AS "memberId",
  "o"."id"::text AS "orgId",
  "o"."name" AS "orgName",
  "m"."role"::text AS "role",
  "m"."memberName" AS "displayName",
  "a"."loginName",
  (
    "a"."memberId" IS NOT NULL
    AND "o"."status" = 1
    AND "m"."status" = 1
    AND "m"."deletedAt" IS NULL
    AND "a"."deletedAt" IS NULL
    AND ("a"."lockedUntil" IS NULL OR "a"."lockedUntil" <= NOW())
  ) AS "available",
  ("a"."passwordHash" IS NOT NULL) AS "hasPassword",
  "a"."passwordUpdatedAt",
  "a"."createdAt",
  "a"."updatedAt"
FROM "public"."SaasOrgMemberAuth" AS "a"
JOIN "public"."SaasOrgMember" AS "m" ON "m"."id" = "a"."memberId"
JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "m"."orgId"
CROSS JOIN "input" AS "i"
WHERE LOWER(BTRIM("a"."loginName")) = "i"."email"
  AND "a"."deletedAt" IS NULL
ORDER BY "accountType", "createdAt";`
	}
	return `
WITH "input" AS (
  SELECT $1::uuid AS "id"
)
SELECT
  'PERSONAL'::text AS "accountType",
  "u"."id"::text AS "identityId",
  "u"."id"::text AS "walletUserId",
  NULL::text AS "memberId",
  NULL::text AS "orgId",
  NULL::text AS "orgName",
  NULL::text AS "role",
  "u"."userName" AS "displayName",
  "u"."email" AS "loginName",
  TRUE AS "available",
  ("u"."passwordHash" IS NOT NULL) AS "hasPassword",
  "u"."passwordUpdatedAt",
  "u"."createdAt",
  "u"."updatedAt"
FROM "public"."User" AS "u"
JOIN "input" AS "i" ON "u"."id" = "i"."id"

UNION ALL

SELECT
  'ORG_WALLET'::text AS "accountType",
  "o"."ownerPbdUserId"::text AS "identityId",
  "o"."ownerPbdUserId"::text AS "walletUserId",
  NULL::text AS "memberId",
  "o"."id"::text AS "orgId",
  "o"."name" AS "orgName",
  NULL::text AS "role",
  "o"."name" AS "displayName",
  NULL::text AS "loginName",
  ("o"."status" = 1) AS "available",
  FALSE AS "hasPassword",
  NULL::timestamptz AS "passwordUpdatedAt",
  "o"."createdAt",
  "o"."updatedAt"
FROM "public"."SaasOrganization" AS "o"
JOIN "input" AS "i" ON "o"."ownerPbdUserId" = "i"."id"

UNION ALL

SELECT
  'ORG_MEMBER'::text AS "accountType",
  "m"."id"::text AS "identityId",
  "o"."ownerPbdUserId"::text AS "walletUserId",
  "m"."id"::text AS "memberId",
  "o"."id"::text AS "orgId",
  "o"."name" AS "orgName",
  "m"."role"::text AS "role",
  "m"."memberName" AS "displayName",
  "a"."loginName",
  (
    "a"."memberId" IS NOT NULL
    AND "o"."status" = 1
    AND "m"."status" = 1
    AND "m"."deletedAt" IS NULL
    AND "a"."deletedAt" IS NULL
    AND ("a"."lockedUntil" IS NULL OR "a"."lockedUntil" <= NOW())
  ) AS "available",
  ("a"."passwordHash" IS NOT NULL) AS "hasPassword",
  "a"."passwordUpdatedAt",
  "a"."createdAt",
  "a"."updatedAt"
FROM "public"."SaasOrgMember" AS "m"
JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "m"."orgId"
LEFT JOIN "public"."SaasOrgMemberAuth" AS "a" ON "a"."memberId" = "m"."id"
JOIN "input" AS "i" ON "m"."id" = "i"."id" OR "m"."pbdUserId" = "i"."id"
ORDER BY "accountType", "createdAt";`
}

func tokenRouterSessionSQL(identifierType string) string {
	return tokenRouterResolvedIdentityCTE(identifierType) + `
SELECT DISTINCT
  "ri"."account_type" AS "accountType",
  "s"."id"::text AS "sessionId",
  "s"."pbdUserId"::text AS "pbdUserId",
  "s"."memberId"::text AS "memberId",
  "s"."orgId"::text AS "orgId",
  "s"."authMethod",
  "s"."status",
  "s"."expiresAt",
  "s"."revokedAt",
  "s"."createdAt",
  "s"."updatedAt"
FROM "public"."UserSession" AS "s"
JOIN "resolved_identity" AS "ri"
  ON "s"."pbdUserId" = "ri"."wallet_user_id"
 AND (
   ("ri"."member_id" IS NULL AND "s"."memberId" IS NULL AND "s"."orgId" IS NULL)
   OR ("s"."memberId" = "ri"."member_id" AND "s"."orgId" = "ri"."org_id")
 )
WHERE (
  NOT $2::boolean
  OR ("s"."status" = 'ACTIVE' AND "s"."revokedAt" IS NULL AND "s"."expiresAt" > NOW())
)
ORDER BY "s"."createdAt" DESC
LIMIT $3;`
}

func tokenRouterWalletSQL(identifierType string) string {
	return tokenRouterResolvedIdentityCTE(identifierType) + `,
"wallet_user" AS (
  SELECT DISTINCT "wallet_user_id" FROM "resolved_identity"
),
"gift" AS (
  SELECT
    "g"."userId",
    COALESCE(SUM("g"."remainingAmount"), 0::bigint) AS "activeGiftBalance"
  FROM "public"."UserGiftBalanceEntry" AS "g"
  JOIN "wallet_user" AS "wu" ON "wu"."wallet_user_id" = "g"."userId"
  WHERE "g"."remainingAmount" > 0
    AND ("g"."expireAt" IS NULL OR "g"."expireAt" > NOW())
  GROUP BY "g"."userId"
)
SELECT
  "wu"."wallet_user_id"::text AS "walletUserId",
  "b"."balance" AS "mainBalanceNanoUsd",
  "b"."balance"::double precision / 1000000000.0 AS "mainBalanceUsd",
  COALESCE("g"."activeGiftBalance", 0) AS "giftBalanceNanoUsd",
  COALESCE("g"."activeGiftBalance", 0)::double precision / 1000000000.0 AS "giftBalanceUsd",
  "b"."creditStatus",
  "b"."creditLimit" AS "creditLimitNanoUsd",
  "b"."creditLimit"::double precision / 1000000000.0 AS "creditLimitUsd",
  "b"."creditUsed" AS "creditUsedNanoUsd",
  "b"."creditUsed"::double precision / 1000000000.0 AS "creditUsedUsd",
  CASE
    WHEN "b"."creditStatus" = 'ENABLED' AND NOT "b"."creditOverdue"
      THEN GREATEST("b"."creditLimit" - "b"."creditUsed", 0)
    ELSE 0
  END AS "creditAvailableNanoUsd",
  CASE
    WHEN "b"."creditStatus" = 'ENABLED' AND NOT "b"."creditOverdue"
      THEN GREATEST("b"."creditLimit" - "b"."creditUsed", 0)::double precision / 1000000000.0
    ELSE 0
  END AS "creditAvailableUsd",
  "b"."creditOverdue",
  "b"."creditOverdueAt",
  "b"."updatedAt"
FROM "wallet_user" AS "wu"
LEFT JOIN "public"."SaasUserBalance" AS "b" ON "b"."userId" = "wu"."wallet_user_id"
LEFT JOIN "gift" AS "g" ON "g"."userId" = "wu"."wallet_user_id"
ORDER BY "walletUserId";`
}

func tokenRouterOrganizationSQL(identifierType string) string {
	return tokenRouterResolvedIdentityCTE(identifierType) + `,
"org" AS (
  SELECT DISTINCT "org_id" FROM "resolved_identity" WHERE "org_id" IS NOT NULL
),
"gift" AS (
  SELECT
    "g"."userId",
    COALESCE(SUM("g"."remainingAmount"), 0::bigint) AS "activeGiftBalance"
  FROM "public"."UserGiftBalanceEntry" AS "g"
  WHERE "g"."remainingAmount" > 0
    AND ("g"."expireAt" IS NULL OR "g"."expireAt" > NOW())
  GROUP BY "g"."userId"
)
SELECT
  "o"."id"::text AS "orgId",
  "o"."name",
  "o"."status" AS "orgStatus",
  "o"."ownerPbdUserId"::text AS "ownerPbdUserId",
  "b"."balance" AS "ownerMainBalanceNanoUsd",
  "b"."balance"::double precision / 1000000000.0 AS "ownerMainBalanceUsd",
  COALESCE("g"."activeGiftBalance", 0) AS "ownerGiftBalanceNanoUsd",
  COALESCE("g"."activeGiftBalance", 0)::double precision / 1000000000.0 AS "ownerGiftBalanceUsd",
  "b"."creditStatus",
  "b"."creditLimit" AS "ownerCreditLimitNanoUsd",
  "b"."creditUsed" AS "ownerCreditUsedNanoUsd",
  CASE
    WHEN "b"."creditStatus" = 'ENABLED' AND NOT "b"."creditOverdue"
      THEN GREATEST("b"."creditLimit" - "b"."creditUsed", 0)
    ELSE 0
  END AS "ownerCreditAvailableNanoUsd",
  "b"."creditOverdue",
  "b"."creditOverdueAt",
  "o"."createdAt",
  "o"."updatedAt"
FROM "org"
JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "org"."org_id"
LEFT JOIN "public"."SaasUserBalance" AS "b" ON "b"."userId" = "o"."ownerPbdUserId"
LEFT JOIN "gift" AS "g" ON "g"."userId" = "o"."ownerPbdUserId"
ORDER BY "o"."createdAt" DESC;`
}

func tokenRouterMemberSQL(identifierType string) string {
	return tokenRouterResolvedIdentityCTE(identifierType) + `,
"org" AS (
  SELECT DISTINCT "org_id" FROM "resolved_identity" WHERE "org_id" IS NOT NULL
),
"target_member" AS (
  SELECT DISTINCT "member_id" FROM "resolved_identity" WHERE "member_id" IS NOT NULL
),
"owner_gift" AS (
  SELECT
    "g"."userId",
    COALESCE(SUM("g"."remainingAmount"), 0::bigint) AS "activeGiftBalance"
  FROM "public"."UserGiftBalanceEntry" AS "g"
  WHERE "g"."remainingAmount" > 0
    AND ("g"."expireAt" IS NULL OR "g"."expireAt" > NOW())
  GROUP BY "g"."userId"
)
SELECT
  "m"."id"::text AS "memberId",
  "m"."orgId"::text AS "orgId",
  "o"."name" AS "orgName",
  "m"."role"::text AS "role",
  "m"."pbdUserId"::text AS "pbdUserId",
  "m"."memberName",
  "a"."loginName",
  "m"."status" AS "memberStatus",
  "m"."deletedAt",
  "a"."deletedAt" AS "authDeletedAt",
  "m"."creditLimit" AS "memberCreditLimitNanoUsd",
  "m"."creditLimit"::double precision / 1000000000.0 AS "memberCreditLimitUsd",
  "m"."remainingCredit" AS "memberRemainingCreditNanoUsd",
  "m"."remainingCredit"::double precision / 1000000000.0 AS "memberRemainingCreditUsd",
  "m"."creditResetMode",
  "m"."creditRollover",
  "m"."creditResetAt",
  COALESCE("b"."balance", 0) + COALESCE("og"."activeGiftBalance", 0) +
    CASE
      WHEN "b"."creditStatus" = 'ENABLED' AND NOT "b"."creditOverdue"
        THEN GREATEST("b"."creditLimit" - "b"."creditUsed", 0)
      ELSE 0
    END AS "ownerCapacityNanoUsd",
  CASE
    WHEN "o"."status" <> 1 OR "m"."status" <> 1 OR "m"."deletedAt" IS NOT NULL THEN 0
    ELSE LEAST(
      "m"."remainingCredit",
      COALESCE("b"."balance", 0) + COALESCE("og"."activeGiftBalance", 0) +
        CASE
          WHEN "b"."creditStatus" = 'ENABLED' AND NOT "b"."creditOverdue"
            THEN GREATEST("b"."creditLimit" - "b"."creditUsed", 0)
          ELSE 0
        END
    )
  END AS "effectiveCapacityNanoUsd",
  "m"."createdAt",
  "m"."updatedAt"
FROM "org"
JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "org"."org_id"
JOIN "target_member" AS "tm" ON TRUE
JOIN "public"."SaasOrgMember" AS "m" ON "m"."id" = "tm"."member_id" AND "m"."orgId" = "o"."id"
LEFT JOIN "public"."SaasOrgMemberAuth" AS "a" ON "a"."memberId" = "m"."id"
LEFT JOIN "public"."SaasUserBalance" AS "b" ON "b"."userId" = "o"."ownerPbdUserId"
LEFT JOIN "owner_gift" AS "og" ON "og"."userId" = "o"."ownerPbdUserId"
ORDER BY
  CASE "m"."role" WHEN 'ORG_SUPER_ADMIN' THEN 1 WHEN 'ORG_ADMIN' THEN 2 ELSE 3 END,
  "m"."createdAt";`
}

func tokenRouterResolvedIdentityCTE(identifierType string) string {
	if identifierType == "email" {
		return `
WITH "input" AS (
  SELECT LOWER(BTRIM($1)) AS "email"
),
"resolved_identity" AS (
  SELECT DISTINCT
    'PERSONAL'::text AS "account_type",
    "u"."id" AS "wallet_user_id",
    NULL::uuid AS "member_id",
    NULL::uuid AS "org_id"
  FROM "public"."User" AS "u"
  CROSS JOIN "input" AS "i"
  WHERE LOWER(BTRIM("u"."email")) = "i"."email"

  UNION ALL

  SELECT DISTINCT
    'ORG'::text AS "account_type",
    "o"."ownerPbdUserId" AS "wallet_user_id",
    "m"."id" AS "member_id",
    "o"."id" AS "org_id"
  FROM "public"."SaasOrgMemberAuth" AS "a"
  JOIN "public"."SaasOrgMember" AS "m" ON "m"."id" = "a"."memberId"
  JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "m"."orgId"
  CROSS JOIN "input" AS "i"
  WHERE LOWER(BTRIM("a"."loginName")) = "i"."email"
    AND "a"."deletedAt" IS NULL
)`
	}
	return `
WITH "input" AS (
  SELECT $1::uuid AS "id"
),
"resolved_identity" AS (
  SELECT DISTINCT
    'PERSONAL'::text AS "account_type",
    "u"."id" AS "wallet_user_id",
    NULL::uuid AS "member_id",
    NULL::uuid AS "org_id"
  FROM "public"."User" AS "u"
  JOIN "input" AS "i" ON "u"."id" = "i"."id"

  UNION ALL

  SELECT DISTINCT
    'ORG_WALLET'::text AS "account_type",
    "o"."ownerPbdUserId" AS "wallet_user_id",
    NULL::uuid AS "member_id",
    "o"."id" AS "org_id"
  FROM "public"."SaasOrganization" AS "o"
  JOIN "input" AS "i" ON "o"."ownerPbdUserId" = "i"."id"

  UNION ALL

  SELECT DISTINCT
    'ORG_MEMBER'::text AS "account_type",
    "o"."ownerPbdUserId" AS "wallet_user_id",
    "m"."id" AS "member_id",
    "o"."id" AS "org_id"
  FROM "public"."SaasOrgMember" AS "m"
  JOIN "public"."SaasOrganization" AS "o" ON "o"."id" = "m"."orgId"
  JOIN "input" AS "i" ON "m"."id" = "i"."id" OR "m"."pbdUserId" = "i"."id"
)`
}

import { NOW } from "./clock";
import type { User } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

/** Small deterministic spread so "last active" values are not all identical. */
const hoursSince = (h: number) =>
  new Date(NOW.getTime() - h * 3_600_000).toISOString();

export const users: User[] = [
  { id: "usr_1", name: "John Doe", email: "john@acme.example", role: "owner", status: "active", teamIds: ["tm_platform"], projectIds: ["prj_1", "prj_2"], createdAt: daysAgo(190), lastActiveAt: hoursSince(0.2), spendMtd: 1840, hasPersonalToken: true },
  { id: "usr_2", name: "Jane Smith", email: "jane@acme.example", role: "admin", status: "active", teamIds: ["tm_platform"], projectIds: ["prj_2", "prj_3"], createdAt: daysAgo(186), lastActiveAt: hoursSince(1.1), spendMtd: 1120, hasPersonalToken: true },
  { id: "usr_3", name: "Priya Sharma", email: "priya.sharma@acme.example", role: "member", status: "active", teamIds: ["tm_platform"], projectIds: ["prj_3"], createdAt: daysAgo(170), lastActiveAt: hoursSince(3.4), spendMtd: 640, hasPersonalToken: false },
  { id: "usr_4", name: "Michael Chen", email: "michael@acme.example", role: "member", status: "active", teamIds: ["tm_platform"], projectIds: ["prj_1"], createdAt: daysAgo(160), lastActiveAt: hoursSince(6.8), spendMtd: 510, hasPersonalToken: true },
  { id: "usr_5", name: "Sarah Wilson", email: "sarah@acme.example", role: "member", status: "invited", teamIds: ["tm_platform"], projectIds: [], createdAt: daysAgo(4), lastActiveAt: hoursSince(0), spendMtd: 0, hasPersonalToken: false },
  { id: "usr_6", name: "Alex Morgan", email: "alex@acme.example", role: "admin", status: "active", teamIds: ["tm_support"], projectIds: ["prj_4", "prj_5"], createdAt: daysAgo(178), lastActiveAt: hoursSince(0.5), spendMtd: 1402, hasPersonalToken: true },
  { id: "usr_7", name: "Jonas Berg", email: "jonas@acme.example", role: "member", status: "active", teamIds: ["tm_support"], projectIds: ["prj_5"], createdAt: daysAgo(166), lastActiveAt: hoursSince(2.2), spendMtd: 880, hasPersonalToken: false },
  { id: "usr_8", name: "Amara Nwosu", email: "amara@acme.example", role: "member", status: "active", teamIds: ["tm_support"], projectIds: ["prj_4"], createdAt: daysAgo(150), lastActiveAt: hoursSince(4.9), spendMtd: 760, hasPersonalToken: false },
  { id: "usr_9", name: "Lucas Ferreira", email: "lucas@acme.example", role: "member", status: "active", teamIds: ["tm_support"], projectIds: ["prj_4", "prj_5"], createdAt: daysAgo(128), lastActiveAt: hoursSince(9.3), spendMtd: 690, hasPersonalToken: true },
  { id: "usr_10", name: "Hannah Kim", email: "hannah@acme.example", role: "member", status: "active", teamIds: ["tm_support"], projectIds: ["prj_5"], createdAt: daysAgo(112), lastActiveAt: hoursSince(14.6), spendMtd: 410, hasPersonalToken: false },
  { id: "usr_11", name: "Omar Haddad", email: "omar@acme.example", role: "viewer", status: "suspended", teamIds: ["tm_support"], projectIds: [], createdAt: daysAgo(140), lastActiveAt: daysAgo(21), spendMtd: 180, hasPersonalToken: true },
  { id: "usr_12", name: "Sofia Marchetti", email: "sofia@acme.example", role: "admin", status: "active", teamIds: ["tm_data"], projectIds: ["prj_6", "prj_7"], createdAt: daysAgo(174), lastActiveAt: hoursSince(0.8), spendMtd: 1620, hasPersonalToken: true },
  { id: "usr_13", name: "Ravi Deshpande", email: "ravi@acme.example", role: "member", status: "active", teamIds: ["tm_data"], projectIds: ["prj_7"], createdAt: daysAgo(156), lastActiveAt: hoursSince(5.1), spendMtd: 810, hasPersonalToken: false },
  { id: "usr_14", name: "Elena Novak", email: "elena@acme.example", role: "member", status: "active", teamIds: ["tm_data"], projectIds: ["prj_8"], createdAt: daysAgo(90), lastActiveAt: hoursSince(11.7), spendMtd: 480, hasPersonalToken: false },
  { id: "usr_15", name: "Chen Wei", email: "chenwei@acme.example", role: "member", status: "deactivated", teamIds: ["tm_data"], projectIds: [], createdAt: daysAgo(130), lastActiveAt: daysAgo(64), spendMtd: 90, hasPersonalToken: false },
  { id: "usr_16", name: "Beatriz Almeida", email: "beatriz@acme.example", role: "admin", status: "active", teamIds: ["tm_legal"], projectIds: ["prj_9", "prj_10"], createdAt: daysAgo(168), lastActiveAt: hoursSince(1.6), spendMtd: 720, hasPersonalToken: true },
  { id: "usr_17", name: "Nathaniel Boyd", email: "nathaniel@acme.example", role: "member", status: "active", teamIds: ["tm_legal"], projectIds: ["prj_10"], createdAt: daysAgo(140), lastActiveAt: hoursSince(7.2), spendMtd: 342, hasPersonalToken: false },
  { id: "usr_18", name: "Ingrid Sørensen", email: "ingrid@acme.example", role: "member", status: "active", teamIds: ["tm_legal"], projectIds: ["prj_9"], createdAt: daysAgo(100), lastActiveAt: hoursSince(19.4), spendMtd: 0, hasPersonalToken: false },
  { id: "usr_19", name: "Marcus Bell", email: "marcus@acme.example", role: "member", status: "active", teamIds: ["tm_marketing"], projectIds: ["prj_11", "prj_12"], createdAt: daysAgo(158), lastActiveAt: hoursSince(2.8), spendMtd: 690, hasPersonalToken: true },
  { id: "usr_20", name: "Yuki Tanaka", email: "yuki@acme.example", role: "member", status: "active", teamIds: ["tm_marketing"], projectIds: ["prj_12"], createdAt: daysAgo(120), lastActiveAt: hoursSince(8.5), spendMtd: 500, hasPersonalToken: false },
  { id: "usr_21", name: "Lucas Meyer", email: "lucasm@acme.example", role: "viewer", status: "active", teamIds: ["tm_marketing"], projectIds: [], createdAt: daysAgo(44), lastActiveAt: daysAgo(3), spendMtd: 0, hasPersonalToken: false },
  { id: "usr_22", name: "Fatima Al-Rashid", email: "fatima@acme.example", role: "admin", status: "active", teamIds: ["tm_security"], projectIds: ["prj_13"], createdAt: daysAgo(162), lastActiveAt: hoursSince(0.3), spendMtd: 410, hasPersonalToken: true },
  { id: "usr_23", name: "Peter Andersson", email: "peter@acme.example", role: "member", status: "active", teamIds: ["tm_security"], projectIds: ["prj_14"], createdAt: daysAgo(96), lastActiveAt: hoursSince(4.1), spendMtd: 230, hasPersonalToken: false },
  { id: "usr_24", name: "Camille Dubois", email: "camille@acme.example", role: "member", status: "active", teamIds: ["tm_security"], projectIds: ["prj_13"], createdAt: daysAgo(72), lastActiveAt: hoursSince(22.8), spendMtd: 0, hasPersonalToken: false },
];

export const currentUser: User = users.find((user) => user.role === "owner") ?? users[0]!;

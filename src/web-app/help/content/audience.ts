// Who a piece of help is mainly for (doc/151-help-module.md). An audience
// only orders help ("Suggested for you" in the panel); it never hides
// any. The template has one audience. A derived app with distinct kinds
// of user lists them here and maps grants to them, for example:
//
//   export const HELP_AUDIENCE_LIST = ["everyone", "admin"] as const;
//   export const HELP_AUDIENCE_BY_GRANT = [
//     { grant: grants.CREATE_USER, audience: "admin" },
//   ];
//
// The first entry whose grant the user holds wins; none is the default.
// tests/help-content.test.ts fails while an audience has no topic.

export const HELP_AUDIENCE_LIST = ["everyone"] as const;

export type HelpAudience = (typeof HELP_AUDIENCE_LIST)[number];

export const DEFAULT_HELP_AUDIENCE: HelpAudience = "everyone";

export const HELP_AUDIENCE_BY_GRANT: {
  grant: number;
  audience: HelpAudience;
}[] = [];

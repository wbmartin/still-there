// GET /helpInsights (handler/help_handler.go, model/help_model.go
// HelpInsightsView): the tenant's help usage over the last `days` days,
// each list most frequent first.
export interface HelpInsightCount {
  // A search text, a help entry id or a tour id.
  key: string;
  count: number;
}

export interface HelpInsights {
  days: number;
  noResultSearchList: HelpInsightCount[];
  entryViewList: HelpInsightCount[];
  tourCompleteList: HelpInsightCount[];
  tourSkipList: HelpInsightCount[];
}

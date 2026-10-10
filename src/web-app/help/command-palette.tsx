import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import {
  Spotlight,
  type SpotlightActionData,
  type SpotlightActionGroupData,
} from "@mantine/spotlight";
import {
  IconArrowRight,
  IconBook,
  IconChartBar,
  IconSearch,
} from "@tabler/icons-react";
import { grants } from "../constants/grants";
import { navItemListFor } from "../constants/nav";
import { ROUTES } from "../constants/routes";
import { useAuthStore } from "../stores/auth-store";
import { useProfileStore } from "../stores/profile-store";
import { useHelpStore } from "../stores/help-store";
import { GLOSSARY, TOPICS, getPageHelp } from "./help-registry";

/**
 * Cmd+K / Ctrl+K: one box to go to a page or find help. The pages are the
 * app's navigation (constants/nav.ts) as the user's grants and the modules
 * on allow, plus
 * Help insights for a user admin; each page's help purpose is its
 * description. The pages check the grants again.
 */
export default function CommandPalette() {
  const navigate = useNavigate();
  const grantList = useAuthStore((state) => state.grantList);
  const openHelp = useHelpStore((state) => state.openHelp);
  const issueReportsEnabled = useProfileStore(
    (state) => state.profile?.issueReportsEnabled === true,
  );

  const actionList = useMemo<SpotlightActionGroupData[]>(() => {
    const go = (path: string) => () => void navigate(path);
    const pageList: SpotlightActionData[] = navItemListFor(grantList, {
      issueReportsEnabled,
    }).map(
      (item) => ({
        id: `nav-${item.route}`,
        label: item.label,
        description: getPageHelp(item.route)?.purpose,
        onClick: go(item.route),
        leftSection: <IconArrowRight size={16} />,
      }),
    );
    if (grantList.includes(grants.READ_USER)) {
      pageList.push({
        id: `nav-${ROUTES.HELP_INSIGHTS}`,
        label: "Help insights",
        description: getPageHelp(ROUTES.HELP_INSIGHTS)?.purpose,
        onClick: go(ROUTES.HELP_INSIGHTS),
        leftSection: <IconChartBar size={16} />,
      });
    }
    const helpList: SpotlightActionData[] = [
      ...GLOSSARY.map((term) => ({
        id: `help-${term.id}`,
        label: term.term,
        description: term.short,
        keywords: term.aliases,
        onClick: () => openHelp(term.id),
        leftSection: <IconBook size={16} />,
      })),
      ...TOPICS.map((topic) => ({
        id: `help-${topic.id}`,
        label: topic.title,
        description: topic.summary,
        onClick: () => openHelp(topic.id),
        leftSection: <IconBook size={16} />,
      })),
    ];
    return [
      { group: "Go to", actions: pageList },
      { group: "Help", actions: helpList },
    ];
  }, [grantList, issueReportsEnabled, navigate, openHelp]);

  return (
    <Spotlight
      actions={actionList}
      shortcut={["mod + K"]}
      limit={12}
      highlightQuery
      nothingFound="Nothing found. Press ? for Help."
      searchProps={{
        leftSection: <IconSearch size={16} />,
        placeholder: "Go to a page or find help",
        "aria-label": "Go to a page or find help",
      }}
    />
  );
}

import { useAuthStore } from "../stores/auth-store";
import type { HelpAudience } from "./content/audience";
import { audienceFromGrants } from "./help-registry";

// The signed-in user's help audience, from the session's grants.
export const useHelpAudience = (): HelpAudience =>
  audienceFromGrants(useAuthStore((state) => state.grantList));

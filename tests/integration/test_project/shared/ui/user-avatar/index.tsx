import { serverData } from "@rstf/shared/ui/user-avatar";

export function UserAvatar() {
  const { name, status } = serverData();
  return (
    <aside data-testid="user-avatar">
      <strong>{name}</strong> <span>({status})</span>
    </aside>
  );
}

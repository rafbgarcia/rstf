import { SSR, type SharedUiUserAvatarSSRProps } from "@rstf/shared/ui/user-avatar";

export const UserAvatar = SSR(function UserAvatar({ name, status }: SharedUiUserAvatarSSRProps) {
  return (
    <aside data-testid="user-avatar">
      <strong>{name}</strong> <span>({status})</span>
    </aside>
  );
});

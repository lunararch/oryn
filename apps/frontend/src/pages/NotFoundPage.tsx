import { Compass } from "lucide-react";
import { Link } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";

export function NotFoundPage() {
  return (
    <EmptyState
      icon={Compass}
      title="This page is off the shelf"
      description="The address does not match a page in Oryn."
      action={
        <Link to="/" className="link-quiet text-sm">
          Return to the dashboard
        </Link>
      }
    />
  );
}

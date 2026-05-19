import { NavLink } from 'react-router-dom';

export function AppNav() {
  return (
    <nav className="flex gap-1 text-sm">
      <NavLink
        to="/"
        end
        className={({ isActive }) =>
          `px-3 py-1.5 rounded-md transition-colors ${
            isActive
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:text-foreground'
          }`
        }
      >
        Identities
      </NavLink>
      <NavLink
        to="/roles"
        className={({ isActive }) =>
          `px-3 py-1.5 rounded-md transition-colors ${
            isActive
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:text-foreground'
          }`
        }
      >
        Roles
      </NavLink>
      <NavLink
        to="/oauth2/clients"
        className={({ isActive }) =>
          `px-3 py-1.5 rounded-md transition-colors ${
            isActive
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:text-foreground'
          }`
        }
      >
        OAuth2 Clients
      </NavLink>
      <NavLink
        to="/oauth2/tokens"
        className={({ isActive }) =>
          `px-3 py-1.5 rounded-md transition-colors ${
            isActive
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:text-foreground'
          }`
        }
      >
        Tokens
      </NavLink>
      <NavLink
        to="/audit-logs"
        className={({ isActive }) =>
          `px-3 py-1.5 rounded-md transition-colors ${
            isActive
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:text-foreground'
          }`
        }
      >
        Audit Log
      </NavLink>
    </nav>
  );
}

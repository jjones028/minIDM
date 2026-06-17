import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import {
  getOAuthClient, listClientRoles, createClientRole, updateClientRole, deleteClientRole,
  listIdentitiesWithRole, assignIdentityToRole, removeIdentityFromRole,
  listGroupsForRole, assignGroupToRole, removeGroupFromRole,
  listClientGroups, createClientGroup, updateClientGroup, deleteClientGroup,
  listGroupMembers, addGroupMember, removeGroupMember,
  listGroupRoles, addRoleToGroup, removeRoleFromGroup,
  getIdentities,
  isUnauthorized,
  type OAuthClient, type ClientRole, type ClientGroup, type RoleMember, type RoleGroup, type Identity,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { AppNav } from '@/components/app-nav';
import { AxiosError } from 'axios';

type Tab = 'roles' | 'groups';

export default function ClientDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [client, setClient] = useState<OAuthClient | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('roles');

  // Roles state
  const [roles, setRoles] = useState<ClientRole[]>([]);
  const [newRoleName, setNewRoleName] = useState('');
  const [newRoleDesc, setNewRoleDesc] = useState('');
  const [expandedRole, setExpandedRole] = useState<string | null>(null);
  const [editingRoleId, setEditingRoleId] = useState<string | null>(null);
  const [editRoleName, setEditRoleName] = useState('');
  const [editRoleDesc, setEditRoleDesc] = useState('');
  const [roleMembers, setRoleMembers] = useState<Record<string, RoleMember[]>>({});
  const [roleGroups, setRoleGroups] = useState<Record<string, RoleGroup[]>>({});

  // Groups state
  const [groups, setGroups] = useState<ClientGroup[]>([]);
  const [newGroupName, setNewGroupName] = useState('');
  const [newGroupDesc, setNewGroupDesc] = useState('');
  const [expandedGroup, setExpandedGroup] = useState<string | null>(null);
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [editGroupName, setEditGroupName] = useState('');
  const [editGroupDesc, setEditGroupDesc] = useState('');
  const [groupMembers, setGroupMembers] = useState<Record<string, RoleMember[]>>({});
  const [groupRoles, setGroupRoles] = useState<Record<string, ClientRole[]>>({});

  // All identities (for assignment pickers)
  const [allIdentities, setAllIdentities] = useState<Identity[]>([]);

  const handleUnauth = useCallback((err: unknown) => {
    if (isUnauthorized(err)) {
      setAuthenticated(false);
      navigate('/login');
      return true;
    }
    return false;
  }, [navigate, setAuthenticated]);

  const fetchRoles = useCallback(async () => {
    if (!id) return;
    try {
      const { data } = await listClientRoles(id);
      setRoles(data ?? []);
    } catch (err) { handleUnauth(err); }
  }, [id, handleUnauth]);

  const fetchGroups = useCallback(async () => {
    if (!id) return;
    try {
      const { data } = await listClientGroups(id);
      setGroups(data ?? []);
    } catch (err) { handleUnauth(err); }
  }, [id, handleUnauth]);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    Promise.all([getOAuthClient(id), listClientRoles(id), listClientGroups(id), getIdentities()])
      .then(([clientRes, rolesRes, groupsRes, identRes]) => {
        if (cancelled) return;
        setClient(clientRes.data);
        setRoles(rolesRes.data ?? []);
        setGroups(groupsRes.data ?? []);
        setAllIdentities(identRes.data ?? []);
      })
      .catch(err => { if (!cancelled) handleUnauth(err); });
    return () => { cancelled = true; };
  }, [id, handleUnauth]);

  // Expand a role: load its members and groups
  const handleExpandRole = async (roleId: string) => {
    if (expandedRole === roleId) {
      setExpandedRole(null);
      return;
    }
    setExpandedRole(roleId);
    if (!id) return;
    try {
      const [membersRes, groupsRes] = await Promise.all([
        listIdentitiesWithRole(id, roleId),
        listGroupsForRole(id, roleId),
      ]);
      setRoleMembers(prev => ({ ...prev, [roleId]: membersRes.data ?? [] }));
      setRoleGroups(prev => ({ ...prev, [roleId]: groupsRes.data ?? [] }));
    } catch (err) { handleUnauth(err); }
  };

  // Expand a group: load its members and roles
  const handleExpandGroup = async (groupId: string) => {
    if (expandedGroup === groupId) {
      setExpandedGroup(null);
      return;
    }
    setExpandedGroup(groupId);
    if (!id) return;
    try {
      const [membersRes, rolesRes] = await Promise.all([
        listGroupMembers(id, groupId),
        listGroupRoles(id, groupId),
      ]);
      setGroupMembers(prev => ({ ...prev, [groupId]: membersRes.data ?? [] }));
      setGroupRoles(prev => ({ ...prev, [groupId]: rolesRes.data ?? [] }));
    } catch (err) { handleUnauth(err); }
  };

  // Role CRUD
  const handleCreateRole = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !newRoleName.trim()) return;
    try {
      await createClientRole(id, newRoleName.trim(), newRoleDesc.trim());
      setNewRoleName('');
      setNewRoleDesc('');
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const startEditRole = (role: ClientRole) => {
    setEditingRoleId(role.id);
    setEditRoleName(role.name);
    setEditRoleDesc(role.description ?? '');
  };

  const handleUpdateRole = async (roleId: string) => {
    if (!id) return;
    try {
      await updateClientRole(id, roleId, editRoleName.trim(), editRoleDesc.trim());
      setEditingRoleId(null);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleDeleteRole = async (roleId: string) => {
    if (!id || !confirm('Delete this role? Identities and groups lose this role.')) return;
    try {
      await deleteClientRole(id, roleId);
      if (expandedRole === roleId) setExpandedRole(null);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  // Role member management
  const handleAssignIdentityToRole = async (roleId: string, identityId: string) => {
    if (!id) return;
    try {
      await assignIdentityToRole(id, roleId, identityId);
      const { data } = await listIdentitiesWithRole(id, roleId);
      setRoleMembers(prev => ({ ...prev, [roleId]: data ?? [] }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleRemoveIdentityFromRole = async (roleId: string, identityId: string) => {
    if (!id) return;
    try {
      await removeIdentityFromRole(id, roleId, identityId);
      setRoleMembers(prev => ({
        ...prev,
        [roleId]: (prev[roleId] ?? []).filter(m => m.id !== identityId),
      }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  // Role group management
  const handleAssignGroupToRole = async (roleId: string, groupId: string) => {
    if (!id) return;
    try {
      await assignGroupToRole(id, roleId, groupId);
      const { data } = await listGroupsForRole(id, roleId);
      setRoleGroups(prev => ({ ...prev, [roleId]: data ?? [] }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleRemoveGroupFromRole = async (roleId: string, groupId: string) => {
    if (!id) return;
    try {
      await removeGroupFromRole(id, roleId, groupId);
      setRoleGroups(prev => ({
        ...prev,
        [roleId]: (prev[roleId] ?? []).filter(g => g.id !== groupId),
      }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  // Group CRUD
  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !newGroupName.trim()) return;
    try {
      await createClientGroup(id, newGroupName.trim(), newGroupDesc.trim());
      setNewGroupName('');
      setNewGroupDesc('');
      fetchGroups();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const startEditGroup = (group: ClientGroup) => {
    setEditingGroupId(group.id);
    setEditGroupName(group.name);
    setEditGroupDesc(group.description ?? '');
  };

  const handleUpdateGroup = async (groupId: string) => {
    if (!id) return;
    try {
      await updateClientGroup(id, groupId, editGroupName.trim(), editGroupDesc.trim());
      setEditingGroupId(null);
      fetchGroups();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleDeleteGroup = async (groupId: string) => {
    if (!id || !confirm('Delete this group? Members lose any roles granted by this group.')) return;
    try {
      await deleteClientGroup(id, groupId);
      if (expandedGroup === groupId) setExpandedGroup(null);
      fetchGroups();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  // Group member management
  const handleAddGroupMember = async (groupId: string, identityId: string) => {
    if (!id) return;
    try {
      await addGroupMember(id, groupId, identityId);
      const { data } = await listGroupMembers(id, groupId);
      setGroupMembers(prev => ({ ...prev, [groupId]: data ?? [] }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleRemoveGroupMember = async (groupId: string, identityId: string) => {
    if (!id) return;
    try {
      await removeGroupMember(id, groupId, identityId);
      setGroupMembers(prev => ({
        ...prev,
        [groupId]: (prev[groupId] ?? []).filter(m => m.id !== identityId),
      }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  // Group role management
  const handleAddRoleToGroup = async (groupId: string, roleId: string) => {
    if (!id) return;
    try {
      await addRoleToGroup(id, groupId, roleId);
      const { data } = await listGroupRoles(id, groupId);
      setGroupRoles(prev => ({ ...prev, [groupId]: data ?? [] }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const handleRemoveRoleFromGroup = async (groupId: string, roleId: string) => {
    if (!id) return;
    try {
      await removeRoleFromGroup(id, groupId, roleId);
      setGroupRoles(prev => ({
        ...prev,
        [groupId]: (prev[groupId] ?? []).filter(r => r.id !== roleId),
      }));
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? (err as Error).message));
    }
  };

  const tabClass = (tab: Tab) =>
    `px-3 py-1.5 rounded-md text-sm transition-colors cursor-pointer ${
      activeTab === tab
        ? 'bg-muted text-foreground font-medium'
        : 'text-muted-foreground hover:text-foreground'
    }`;

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-5xl mx-auto space-y-8">
        <header className="flex items-start justify-between">
          <div className="space-y-2">
            <h1 className="text-5xl font-extrabold tracking-tight font-heading">minidm</h1>
            <AppNav />
          </div>
          <Button variant="outline" onClick={() => { setAuthenticated(false); navigate('/login'); }}>
            Sign Out
          </Button>
        </header>

        <div className="space-y-1">
          <Link to="/oauth2/clients" className="text-sm text-muted-foreground hover:text-foreground">
            ← OAuth2 Clients
          </Link>
          {client && (
            <h2 className="text-2xl font-bold">{client.name}</h2>
          )}
          {client?.description && (
            <p className="text-muted-foreground text-sm">{client.description}</p>
          )}
          {client && (
            <code className="text-xs bg-muted rounded px-1.5 py-0.5">{client.client_id}</code>
          )}
        </div>

        {/* Tabs */}
        <div className="flex gap-1 border-b pb-0">
          <button className={tabClass('roles')} onClick={() => setActiveTab('roles')}>Roles</button>
          <button className={tabClass('groups')} onClick={() => setActiveTab('groups')}>Groups</button>
        </div>

        {/* ---- Roles Tab ---- */}
        {activeTab === 'roles' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Create Role</CardTitle>
                <CardDescription>
                  Define a role for this client. Identities assigned this role will receive it in the <code>roles</code> JWT claim.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCreateRole} className="flex gap-2 items-end flex-wrap">
                  <div className="grid gap-1 flex-1 min-w-32">
                    <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Name</label>
                    <Input
                      placeholder="admin"
                      value={newRoleName}
                      onChange={e => setNewRoleName(e.target.value)}
                      required
                      className="h-8"
                    />
                  </div>
                  <div className="grid gap-1 flex-1 min-w-48">
                    <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Description</label>
                    <Input
                      placeholder="Optional description"
                      value={newRoleDesc}
                      onChange={e => setNewRoleDesc(e.target.value)}
                      className="h-8"
                    />
                  </div>
                  <Button type="submit" size="sm">Create Role</Button>
                </form>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Roles</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="pl-6">Name</TableHead>
                      <TableHead>Description</TableHead>
                      <TableHead className="w-32" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {roles.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={3} className="pl-6 text-muted-foreground">
                          No roles defined yet.
                        </TableCell>
                      </TableRow>
                    )}
                    {roles.map(role => (
                      <>
                        <TableRow key={role.id}>
                          {editingRoleId === role.id ? (
                            <TableCell colSpan={3} className="pl-6">
                              <div className="flex gap-2 items-center flex-wrap">
                                <Input
                                  value={editRoleName}
                                  onChange={e => setEditRoleName(e.target.value)}
                                  className="h-7 w-40"
                                />
                                <Input
                                  value={editRoleDesc}
                                  onChange={e => setEditRoleDesc(e.target.value)}
                                  placeholder="Description"
                                  className="h-7 flex-1 min-w-40"
                                />
                                <Button size="sm" className="h-7" onClick={() => handleUpdateRole(role.id)}>Save</Button>
                                <Button size="sm" variant="ghost" className="h-7" onClick={() => setEditingRoleId(null)}>Cancel</Button>
                              </div>
                            </TableCell>
                          ) : (
                            <>
                              <TableCell className="pl-6 font-medium">{role.name}</TableCell>
                              <TableCell className="text-muted-foreground text-sm">{role.description ?? ''}</TableCell>
                              <TableCell>
                                <div className="flex gap-1 justify-end pr-4">
                                  <Button
                                    size="sm" variant="ghost" className="h-7 text-xs"
                                    onClick={() => handleExpandRole(role.id)}
                                  >
                                    {expandedRole === role.id ? 'Close' : 'Members'}
                                  </Button>
                                  <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={() => startEditRole(role)}>Edit</Button>
                                  <Button
                                    size="sm" variant="ghost" className="h-7 text-xs text-destructive hover:text-destructive"
                                    onClick={() => handleDeleteRole(role.id)}
                                  >
                                    Delete
                                  </Button>
                                </div>
                              </TableCell>
                            </>
                          )}
                        </TableRow>
                        {expandedRole === role.id && (
                          <TableRow key={`${role.id}-expand`}>
                            <TableCell colSpan={3} className="pl-6 pr-6 pb-6">
                              <RoleExpanded
                                clientDbId={id!}
                                role={role}
                                members={roleMembers[role.id] ?? []}
                                assignedGroups={roleGroups[role.id] ?? []}
                                allIdentities={allIdentities}
                                allGroups={groups}
                                onAssignIdentity={identityId => handleAssignIdentityToRole(role.id, identityId)}
                                onRemoveIdentity={identityId => handleRemoveIdentityFromRole(role.id, identityId)}
                                onAssignGroup={groupId => handleAssignGroupToRole(role.id, groupId)}
                                onRemoveGroup={groupId => handleRemoveGroupFromRole(role.id, groupId)}
                              />
                            </TableCell>
                          </TableRow>
                        )}
                      </>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        )}

        {/* ---- Groups Tab ---- */}
        {activeTab === 'groups' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Create Group</CardTitle>
                <CardDescription>
                  Groups let you assign multiple identities to roles in bulk. Roles granted to a group flow to all members.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCreateGroup} className="flex gap-2 items-end flex-wrap">
                  <div className="grid gap-1 flex-1 min-w-32">
                    <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Name</label>
                    <Input
                      placeholder="billing-team"
                      value={newGroupName}
                      onChange={e => setNewGroupName(e.target.value)}
                      required
                      className="h-8"
                    />
                  </div>
                  <div className="grid gap-1 flex-1 min-w-48">
                    <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Description</label>
                    <Input
                      placeholder="Optional description"
                      value={newGroupDesc}
                      onChange={e => setNewGroupDesc(e.target.value)}
                      className="h-8"
                    />
                  </div>
                  <Button type="submit" size="sm">Create Group</Button>
                </form>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Groups</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="pl-6">Name</TableHead>
                      <TableHead>Description</TableHead>
                      <TableHead className="w-32" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {groups.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={3} className="pl-6 text-muted-foreground">
                          No groups defined yet.
                        </TableCell>
                      </TableRow>
                    )}
                    {groups.map(group => (
                      <>
                        <TableRow key={group.id}>
                          {editingGroupId === group.id ? (
                            <TableCell colSpan={3} className="pl-6">
                              <div className="flex gap-2 items-center flex-wrap">
                                <Input
                                  value={editGroupName}
                                  onChange={e => setEditGroupName(e.target.value)}
                                  className="h-7 w-40"
                                />
                                <Input
                                  value={editGroupDesc}
                                  onChange={e => setEditGroupDesc(e.target.value)}
                                  placeholder="Description"
                                  className="h-7 flex-1 min-w-40"
                                />
                                <Button size="sm" className="h-7" onClick={() => handleUpdateGroup(group.id)}>Save</Button>
                                <Button size="sm" variant="ghost" className="h-7" onClick={() => setEditingGroupId(null)}>Cancel</Button>
                              </div>
                            </TableCell>
                          ) : (
                            <>
                              <TableCell className="pl-6 font-medium">{group.name}</TableCell>
                              <TableCell className="text-muted-foreground text-sm">{group.description ?? ''}</TableCell>
                              <TableCell>
                                <div className="flex gap-1 justify-end pr-4">
                                  <Button
                                    size="sm" variant="ghost" className="h-7 text-xs"
                                    onClick={() => handleExpandGroup(group.id)}
                                  >
                                    {expandedGroup === group.id ? 'Close' : 'Manage'}
                                  </Button>
                                  <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={() => startEditGroup(group)}>Edit</Button>
                                  <Button
                                    size="sm" variant="ghost" className="h-7 text-xs text-destructive hover:text-destructive"
                                    onClick={() => handleDeleteGroup(group.id)}
                                  >
                                    Delete
                                  </Button>
                                </div>
                              </TableCell>
                            </>
                          )}
                        </TableRow>
                        {expandedGroup === group.id && (
                          <TableRow key={`${group.id}-expand`}>
                            <TableCell colSpan={3} className="pl-6 pr-6 pb-6">
                              <GroupExpanded
                                group={group}
                                members={groupMembers[group.id] ?? []}
                                assignedRoles={groupRoles[group.id] ?? []}
                                allIdentities={allIdentities}
                                allRoles={roles}
                                onAddMember={identityId => handleAddGroupMember(group.id, identityId)}
                                onRemoveMember={identityId => handleRemoveGroupMember(group.id, identityId)}
                                onAddRole={roleId => handleAddRoleToGroup(group.id, roleId)}
                                onRemoveRole={roleId => handleRemoveRoleFromGroup(group.id, roleId)}
                              />
                            </TableCell>
                          </TableRow>
                        )}
                      </>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
}

// ---- Sub-components ----

function RoleExpanded({
  members, assignedGroups, allIdentities, allGroups,
  onAssignIdentity, onRemoveIdentity, onAssignGroup, onRemoveGroup,
}: {
  clientDbId: string;
  role: ClientRole;
  members: RoleMember[];
  assignedGroups: RoleGroup[];
  allIdentities: Identity[];
  allGroups: ClientGroup[];
  onAssignIdentity: (identityId: string) => void;
  onRemoveIdentity: (identityId: string) => void;
  onAssignGroup: (groupId: string) => void;
  onRemoveGroup: (groupId: string) => void;
}) {
  const assignedMemberIds = new Set(members.map(m => m.id));
  const assignedGroupIds = new Set(assignedGroups.map(g => g.id));
  const availableIdentities = allIdentities.filter(i => !assignedMemberIds.has(i.id));
  const availableGroups = allGroups.filter(g => !assignedGroupIds.has(g.id));

  return (
    <div className="grid md:grid-cols-2 gap-6 pt-2">
      <div className="space-y-3">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Direct Members</p>
        {availableIdentities.length > 0 && (
          <select
            className="w-full h-7 text-sm rounded-md border border-input bg-background px-2 focus:outline-none focus:ring-1 focus:ring-ring"
            defaultValue=""
            onChange={e => { if (e.target.value) onAssignIdentity(e.target.value); e.target.value = ''; }}
          >
            <option value="" disabled>Add identity…</option>
            {availableIdentities.map(i => (
              <option key={i.id} value={i.id}>{i.email}</option>
            ))}
          </select>
        )}
        {members.length === 0
          ? <p className="text-sm text-muted-foreground">No direct members.</p>
          : <div className="space-y-1">
              {members.map(m => (
                <div key={m.id} className="flex items-center justify-between text-sm py-0.5">
                  <span>{m.email}</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => onRemoveIdentity(m.id)}>
                    Remove
                  </Button>
                </div>
              ))}
            </div>
        }
      </div>
      <div className="space-y-3">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Groups with this Role</p>
        {availableGroups.length > 0 && (
          <select
            className="w-full h-7 text-sm rounded-md border border-input bg-background px-2 focus:outline-none focus:ring-1 focus:ring-ring"
            defaultValue=""
            onChange={e => { if (e.target.value) onAssignGroup(e.target.value); e.target.value = ''; }}
          >
            <option value="" disabled>Add group…</option>
            {availableGroups.map(g => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        )}
        {assignedGroups.length === 0
          ? <p className="text-sm text-muted-foreground">No groups assigned.</p>
          : <div className="space-y-1">
              {assignedGroups.map(g => (
                <div key={g.id} className="flex items-center justify-between text-sm py-0.5">
                  <span>{g.name}</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => onRemoveGroup(g.id)}>
                    Remove
                  </Button>
                </div>
              ))}
            </div>
        }
      </div>
    </div>
  );
}

function GroupExpanded({
  members, assignedRoles, allIdentities, allRoles,
  onAddMember, onRemoveMember, onAddRole, onRemoveRole,
}: {
  group: ClientGroup;
  members: RoleMember[];
  assignedRoles: ClientRole[];
  allIdentities: Identity[];
  allRoles: ClientRole[];
  onAddMember: (identityId: string) => void;
  onRemoveMember: (identityId: string) => void;
  onAddRole: (roleId: string) => void;
  onRemoveRole: (roleId: string) => void;
}) {
  const memberIds = new Set(members.map(m => m.id));
  const assignedRoleIds = new Set(assignedRoles.map(r => r.id));
  const availableIdentities = allIdentities.filter(i => !memberIds.has(i.id));
  const availableRoles = allRoles.filter(r => !assignedRoleIds.has(r.id));

  return (
    <div className="grid md:grid-cols-2 gap-6 pt-2">
      <div className="space-y-3">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Members</p>
        {availableIdentities.length > 0 && (
          <select
            className="w-full h-7 text-sm rounded-md border border-input bg-background px-2 focus:outline-none focus:ring-1 focus:ring-ring"
            defaultValue=""
            onChange={e => { if (e.target.value) onAddMember(e.target.value); e.target.value = ''; }}
          >
            <option value="" disabled>Add identity…</option>
            {availableIdentities.map(i => (
              <option key={i.id} value={i.id}>{i.email}</option>
            ))}
          </select>
        )}
        {members.length === 0
          ? <p className="text-sm text-muted-foreground">No members.</p>
          : <div className="space-y-1">
              {members.map(m => (
                <div key={m.id} className="flex items-center justify-between text-sm py-0.5">
                  <span>{m.email}</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => onRemoveMember(m.id)}>
                    Remove
                  </Button>
                </div>
              ))}
            </div>
        }
      </div>
      <div className="space-y-3">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Roles Granted</p>
        {availableRoles.length > 0 && (
          <select
            className="w-full h-7 text-sm rounded-md border border-input bg-background px-2 focus:outline-none focus:ring-1 focus:ring-ring"
            defaultValue=""
            onChange={e => { if (e.target.value) onAddRole(e.target.value); e.target.value = ''; }}
          >
            <option value="" disabled>Add role…</option>
            {availableRoles.map(r => (
              <option key={r.id} value={r.id}>{r.name}</option>
            ))}
          </select>
        )}
        {assignedRoles.length === 0
          ? <p className="text-sm text-muted-foreground">No roles assigned.</p>
          : <div className="space-y-1">
              {assignedRoles.map(r => (
                <div key={r.id} className="flex items-center justify-between text-sm py-0.5">
                  <span className="font-mono">{r.name}</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive hover:text-destructive" onClick={() => onRemoveRole(r.id)}>
                    Remove
                  </Button>
                </div>
              ))}
            </div>
        }
      </div>
    </div>
  );
}

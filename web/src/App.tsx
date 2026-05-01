import { useState, useEffect } from 'react';
import { getIdentities, registerIdentity, type Identity } from './api';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AxiosError } from 'axios';

function App() {
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const fetchIdentities = async () => {
    try {
      const { data } = await getIdentities();
      setIdentities(data || []);
    } catch (error) {
      console.error('Failed to fetch identities', error);
    }
  };

  useEffect(() => {
    let ignore = false;
    const load = async () => {
      try {
        const { data } = await getIdentities();
        if (!ignore) {
          setIdentities(data || []);
        }
      } catch (error) {
        if (!ignore) {
          console.error('Failed to fetch identities', error);
        }
      }
    };
    load();
    return () => {
      ignore = true;
    };
  }, []);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await registerIdentity({ email, password });
      setEmail('');
      setPassword('');
      fetchIdentities();
    } catch (error) {
      const axiosError = error as AxiosError<string>;
      alert('Registration failed: ' + (axiosError.response?.data || axiosError.message));
    }
  };

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-5xl mx-auto space-y-10">
        <header className="space-y-2">
          <h1 className="text-5xl font-extrabold tracking-tight font-heading">Identity Hub</h1>
          <p className="text-lg text-muted-foreground">Manage your secure identities with ease and precision.</p>
        </header>
        
        <Card>
          <CardHeader>
            <CardTitle>Register Identity</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleRegister} className="flex flex-col md:flex-row gap-4 items-end">
              <div className="grid w-full gap-1.5">
                <label className="text-sm font-medium leading-none">Email Address</label>
                <Input 
                  type="email" 
                  placeholder="name@company.com" 
                  value={email} 
                  onChange={e => setEmail(e.target.value)} 
                  required 
                />
              </div>
              <div className="grid w-full gap-1.5">
                <label className="text-sm font-medium leading-none">Password</label>
                <Input 
                  type="password" 
                  placeholder="••••••••" 
                  value={password} 
                  onChange={e => setPassword(e.target.value)} 
                  required 
                />
              </div>
              <Button type="submit" className="w-full md:w-auto">
                Create Identity
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Identity Registry</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Email</TableHead>
                  <TableHead>Subject ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {identities.map(id => (
                  <TableRow key={id.id}>
                    <TableCell className="pl-6 font-medium">{id.email}</TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">{id.subject_id}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default App;

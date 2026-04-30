import { useState, useEffect } from 'react';
import { getIdentities, registerIdentity } from './api';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

function App() {
  const [identities, setIdentities] = useState([]);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  useEffect(() => {
    fetchIdentities();
  }, []);

  const fetchIdentities = async () => {
    try {
      const { data } = await getIdentities();
      setIdentities(data || []);
    } catch (error) {
      console.error('Failed to fetch identities', error);
    }
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    try {
      await registerIdentity({ email, password });
      setEmail('');
      setPassword('');
      fetchIdentities();
    } catch (error) {
      alert('Registration failed: ' + (error.response?.data || error.message));
    }
  };

  return (
    <div className="min-h-screen bg-background text-foreground selection:bg-blue-500/30 p-4 md:p-12">
      <div className="max-w-5xl mx-auto space-y-10">
        <header className="space-y-2">
          <h1 className="text-5xl font-extrabold tracking-tight text-white">Identity Hub</h1>
          <p className="text-lg text-muted-foreground">Manage your secure identities with ease and precision.</p>
        </header>
        
        <Card className="border-border/50 bg-card/50 backdrop-blur-sm shadow-2xl shadow-blue-500/5">
          <CardHeader className="pb-4">
            <CardTitle className="text-xl font-semibold text-white">Register Identity</CardTitle>
          </CardHeader>
            <CardContent className="p-6">
            <form onSubmit={handleRegister} className="flex flex-col md:flex-row gap-6 items-end">
              <div className="w-full md:w-80 flex-shrink-0 space-y-2">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest pl-0.5">Email Address</label>
                <Input 
                  type="email" 
                  placeholder="name@company.com" 
                  value={email} 
                  onChange={e => setEmail(e.target.value)} 
                  required 
                  className="bg-background/50 border-border/50 focus:border-blue-500 transition-colors h-11 w-full"
                />
              </div>
              <div className="w-full md:w-80 flex-shrink-0 space-y-2">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest pl-0.5">Password</label>
                <Input 
                  type="password" 
                  placeholder="••••••••" 
                  value={password} 
                  onChange={e => setPassword(e.target.value)} 
                  required 
                  className="bg-background/50 border-border/50 focus:border-blue-500 transition-colors h-11 w-full"
                />
              </div>
              <Button 
                type="submit" 
                className="h-11 px-8 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-lg shadow-lg shadow-blue-900/20 transition-all hover:-translate-y-0.5 active:translate-y-0 w-full md:w-auto flex-shrink-0"
              >
                Create Identity
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card className="border-border/50 bg-card/50 backdrop-blur-sm shadow-xl">
          <CardHeader>
            <CardTitle className="text-xl font-semibold text-white">Identity Registry</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50 hover:bg-transparent">
                  <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground pl-6">Email</TableHead>
                  <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Subject ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {identities.map(id => (
                  <TableRow key={id.id} className="border-border/50 hover:bg-blue-500/5 transition-colors">
                    <TableCell className="pl-6 font-medium text-gray-200">{id.email}</TableCell>
                    <TableCell className="text-blue-400/80 font-mono text-sm">{id.subject_id}</TableCell>
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

// Mock generated module for testing (simulates codegen output)
export let Name: string = "";
export let Count: number = 0;

export function __setServerData(data: Record<string, any>) {
  Name = data.Name ?? "";
  Count = data.Count ?? 0;
}

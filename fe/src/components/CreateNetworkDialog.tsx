import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Plus } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import type { ClusterNetwork } from "@/lib/types";

const formSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(63, "Name must be at most 63 characters"),
  cidr: z
    .string()
    .regex(
      /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/,
      "Must be a CIDR like 10.10.0.0/24",
    )
    .refine((v) => {
      const prefix = Number(v.split("/")[1]);
      return prefix >= 8 && prefix <= 29;
    }, "Prefix length must be between /8 and /29"),
});

type FormValues = z.infer<typeof formSchema>;

interface CreateNetworkDialogProps {
  onSuccess?: (network: ClusterNetwork) => void;
}

export function CreateNetworkDialog({ onSuccess }: CreateNetworkDialogProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "", cidr: "" },
  });

  async function onSubmit(values: FormValues) {
    setIsLoading(true);
    try {
      const result = await api.post<{ network: ClusterNetwork }>(
        "/api/v1/networks",
        values,
      );
      setOpen(false);
      form.reset();
      onSuccess?.(result.network);
    } catch (err) {
      form.setError("root", {
        message:
          err instanceof ApiError ? err.message : "Failed to create network",
      });
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) form.reset();
      }}
    >
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Network
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Cluster Network</DialogTitle>
          <DialogDescription>
            Creates an Incus bridge network that clusters can be launched
            onto.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <Alert variant="destructive">
                <AlertDescription>
                  {form.formState.errors.root.message}
                </AlertDescription>
              </Alert>
            )}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Network Name</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="e.g., prod-net"
                      disabled={isLoading}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="cidr"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>CIDR</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="10.10.0.0/24"
                      disabled={isLoading}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    An IPv4 network address (no host bits), prefix between
                    /8 and /29.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setOpen(false)}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? "Creating..." : "Create Network"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

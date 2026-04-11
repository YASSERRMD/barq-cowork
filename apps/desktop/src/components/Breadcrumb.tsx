import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";

interface BreadcrumbItem {
  label: string;
  to?: string;
}

export function Breadcrumb({ items }: { items: BreadcrumbItem[] }) {
  return (
    <nav style={{ display: "flex", alignItems: "center", gap: 4 }}>
      {items.map((item, i) => (
        <span key={i} style={{ display: "flex", alignItems: "center", gap: 4 }}>
          {i > 0 && <ChevronRight size={12} color="#40404f" />}
          {item.to ? (
            <Link
              to={item.to}
              style={{ fontSize: 12, color: "#50505f", textDecoration: "none", transition: "color 120ms" }}
              onMouseEnter={(e) => ((e.target as HTMLElement).style.color = "#9090a0")}
              onMouseLeave={(e) => ((e.target as HTMLElement).style.color = "#50505f")}
            >
              {item.label}
            </Link>
          ) : (
            <span style={{ fontSize: 12, color: "#7a7a90" }}>{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}

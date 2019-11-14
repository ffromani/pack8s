# pack8s kubevirtci provider status

kubevirtci `56f69bb5867db7517f70a0787b32570a861e124a`

## Supported providers:

| Provider          | Run           | Provisioning  | Notes              |
| ----------------- | ------------- | ------------- | ------------------ |
| k8s-1.14.6        | *Yes*         | Planned       |                    |
| k8s-1.15.1        | In progress   | Planned       |                    |
| k8s-genie-1.11.1  | Not yet       | N/A           |                    |
| k8s-multus-1.13.3 | Not yet       | N/A           |                    |
| okd-4.1           | *Yes*         | N/A           |                    |
| okd-4.3           | In progress   | Planned       |                    |
| os-3.11.0         | Not yet       | N/A           |                    |
| os-3.11.0-crio    | Not yet       | N/A           |                    |
| os-3.11.0-multus  | Not yet       | N/A           |                    |

Key:
- Yes: works like gocli, no regression known
- No: something's broken, see notes
- N/A: we don't plan to implement this
- In progress: the team started working on it, still WIP
- Not yet: planned for the near future, work has not begun yet
- Planned: planned for the far future, blocked by something else (likely the "not yet" queue)

## Unsupported providers:

* local
* k8s-1.11.0
* k8s-1.13.3
* kind
* kind-k8s-1.14.2
* kind-k8s-sriov-1.14.2
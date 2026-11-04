import { Link } from 'react-router-dom'
import { Card, ErrorNote, Loading, PageTitle, errorMessage } from '../components'
import { useConversations } from '../query_social'

export function ChatPage() {
  const query = useConversations()

  if (query.isPending) return <Loading label="Loading your chats…" />
  if (query.isError) return <ErrorNote message={errorMessage(query.error)} />

  return (
    <>
      <PageTitle
        title="Chats"
        sub="Only mutual-interest matches can message you. New messages arrive every few seconds."
      />
      {query.data.conversations.length === 0 ? (
        <Card>
          <p className="text-[15px] text-sage">
            No chats yet. Express interest in a match — once it&rsquo;s mutual, you can talk here.
          </p>
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {query.data.conversations.map((c) => (
            <Link key={c.peer_id} to={`/chat/${c.peer_id}`}>
              <Card>
                <div className="flex items-baseline justify-between gap-3">
                  <p className="font-semibold text-ink">
                    {c.peer_name}
                    {c.unread_count > 0 && (
                      <span className="ml-2 inline-flex min-w-5 items-center justify-center rounded-full bg-terracotta px-1.5 py-0.5 text-[11px] font-bold text-white">
                        {c.unread_count}
                      </span>
                    )}
                  </p>
                  <p className="shrink-0 text-xs tabular-nums text-fog">
                    {c.last_at.slice(0, 16).replace('T', ' ')}
                  </p>
                </div>
                <p
                  className={`mt-1 line-clamp-1 text-sm ${c.unread_count > 0 ? 'font-semibold text-ink' : 'text-sage'}`}
                >
                  {c.last_from_me ? 'You: ' : ''}
                  {c.last_body}
                </p>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </>
  )
}
